package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	// Server related stuff
	"net/http"

	"github.com/gbrlsnchs/jwt/v3"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"

	// Auth stuff
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/google"

	// DB stuff
	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	// Other stuff
	"github.com/go-playground/validator"
	"github.com/jkomyno/nanoid"
	_ "github.com/joho/godotenv/autoload"
)

// PublicToPrivateUserID gets the database entry id of a user from database that
// corresponds to a specific public id.
func PublicToPrivateUserID(db *sqlx.DB, PublicID string) (StructID) {
	userIDQuery := sq.Select("id").From("users").Where(sq.Eq{"public_id": PublicID})
	userIDQueryString, userIDQueryStringArgs, err := userIDQuery.ToSql()
	if err != nil {
		log.Fatalln(err.Error())
	}

	user := StructID{}
	if err := db.Get(&user, userIDQueryString, userIDQueryStringArgs...); err != nil {
		log.Fatalln(err.Error())
	}

	return user
}

var hs = jwt.NewHS256([]byte("jdnfksdmfksd"))

// RefreshToken extends tokens current expiration time for another hour. This is
// done so if user uses the webapp, their token won't expire until they stop
// using it for an hour.
func RefreshToken(payload JWTPayload) (string, bool) {
	now := time.Now()
	if (payload.ExpirationTime.Time.Unix() < now.Unix()) {
		payload.ExpirationTime = jwt.NumericDate(now.Add(time.Hour))

		token, err := jwt.Sign(payload, hs)
		if err != nil {
			return "", false
		}

		return string(token), true
	}

	return "", false
}

// TokenVerificationMiddleware verifies token sent via request in the cookie and
// checks if the user exists in the database. Afther that adds user id as a
// property inside request context.
func TokenVerificationMiddleware(db *sqlx.DB) gin.HandlerFunc {
	return func (ctx *gin.Context) {
		token, err := ctx.Cookie("token")
		if err != nil {
			switch err {
			case http.ErrNoCookie:
				ctx.String(http.StatusUnauthorized, "No authorization token cookie found!")
				break
			default:
				ctx.String(http.StatusInternalServerError, err.Error())
			}
			ctx.Abort()
			return
		}

		var payload JWTPayload

		now := time.Now()
		expValidator := jwt.ExpirationTimeValidator(now)
		validatePayload := jwt.ValidatePayload(&payload.Payload, expValidator)

		_, err = jwt.Verify([]byte(token), hs, &payload, validatePayload)
		if err != nil {
			switch err {
			case jwt.ErrExpValidation:
				ctx.String(http.StatusUnauthorized, "The token has expired!")
				break
			default:
				ctx.String(http.StatusInternalServerError, err.Error())
			}
			ctx.Abort()
			return
		}

		// Checking if the user acutally exists. If not, send a cute message.
		userNameQuery := sq.Select("id").From("users").Where(sq.Eq{"public_id": payload.UserID})
		userNameQueryString, userNameQueryStringArgs, err := userNameQuery.ToSql()
		
		var user StructID
		if err := db.Get(&user, userNameQueryString, userNameQueryStringArgs...); err != nil {
			switch err {
			case sql.ErrNoRows:
				ctx.String(http.StatusUnauthorized, "Hey you! You are not supposed to be here! Please go away!")
				break
			default:
				ctx.String(http.StatusInternalServerError, err.Error())
			}
			ctx.Abort()
			return
		}

		refreshedToken, refreshedTokenGenerated := RefreshToken(payload)
		if refreshedTokenGenerated {
			ctx.SetCookie("token", refreshedToken, 3600, "/", "localhost", false, true)
		}

		ctx.Set("userID", payload.UserID)
		ctx.Next()
	}
}

// CreateToken creates a JWT from user structure specified as a parameter.
func CreateToken(user goth.User) (string, error) {
	uuid, err := nanoid.Nanoid()
	if err != nil {
		return "", err
	}
	now := time.Now()
	payload := JWTPayload{
		Payload: jwt.Payload{
			Issuer: "receiptsarchive",
			Subject: user.UserID,
			ExpirationTime: jwt.NumericDate(now.Add(time.Hour)),
			IssuedAt: jwt.NumericDate(now),
			JWTID: uuid,
		},
		UserID: user.UserID,
	}

	token, err := jwt.Sign(payload, hs)
	if err != nil {
		return "", err
	}

	return string(token), nil
}

// GetUserID get the user id from specified context. It's literarly used just
// so I can write one line instead of two.
func GetUserID(ctx *gin.Context) (string, bool) {
	userID, userIDExists := ctx.Get("userID")
	return userID.(string), userIDExists
}

func main() {
	router := gin.Default()
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = strings.Split(os.Getenv("ALLOW_ORIGINS"), ",")
	corsConfig.AllowCredentials = true
	router.Use(cors.New(corsConfig))

	db, err := generateDatabase()
	if err != nil {
		fmt.Println("Failed to connect to the database!")
		log.Fatalln(err.Error())
		return
	}

	gothic.Store = cookie.NewStore([]byte(os.Getenv("COOKIE_SECRET")))
	goth.UseProviders(google.New(os.Getenv("GOOGLE_OAUTH_CLIENT_KEY"), os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"), os.Getenv("GOOGLE_OAUTH_CALLBACK_URL")))

	v := validator.New()

	auth := router.Group("/auth")
	{
		auth.GET("", AuthHandler(db))

		auth.GET("/callback", AuthCallbackHandler(db))
	}

	locations := router.Group("/locations")
	locations.Use(TokenVerificationMiddleware(db))
	{
		// Get list of locations (query available)
		locations.GET("", GetLocationHandler(db))

		// Add new location
		locations.POST("", PostLocationHandler(db))

		// Update location
		locations.PUT("", PutLocationHandler(db, v))

		// Delete location
		locations.DELETE("", DeleteLocationHandler(db, v))
	}

	items := router.Group("/items")
	items.Use(TokenVerificationMiddleware(db))
	{
		// Get list of items (query available)
		items.GET("", GetItemsHandler(db))

		// Get list of items from a specific receipt
		items.GET("/inreceipt/:id", GetItemsInReceiptHandler(db))

		// Add new item
		items.POST("", PostItemsHandler(db, v))

		// Add item to receipts
		items.POST("/inreceipt", PostItemsInReceiptHandler(db, v))

		// Update item
		items.PUT("", PutItemsHandler(db, v))

		// Update item from specific receipt
		items.PUT("/inreceipt", PutItemsInReceiptHandler(db))

		// Delete item
		items.DELETE("", DeleteItemsHandler(db, v))

		// Delete item from receipt
		items.DELETE("/inreceipt", DeleteItemsInReceiptHandler(db, v))
	}

	receipts := router.Group("/receipts")
	receipts.Use(TokenVerificationMiddleware(db))
	{
		// Get list of receipts (query available)
		receipts.GET("", GetReceiptsHandler(db))

		// Add new receipt
		receipts.POST("", PostReceiptsHandler(db, v))

		// Update receipt
		receipts.PUT("", PutReceiptsHandler(db, v))

		// Delete receipt
		receipts.DELETE("", DeleteReceiptsHandler(db, v))
	}

	router.Run(":" + os.Getenv("PORT"))
}

package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	// Server related stuff

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
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
	_ "github.com/joho/godotenv/autoload"
)

// PublicToPrivateUserID gets the database entry id of a user from database that
// corresponds to a specific public id.
func PublicToPrivateUserID(db *sqlx.DB, PublicID string) StructID {
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

// GetUserID get the user id from specified context. It's literarly used just
// so I can write one line instead of two.
func GetUserID(ctx *gin.Context) (string, bool) {
	userID, userIDExists := ctx.Get("userID")
	return userID.(string), userIDExists
}

func main() {
	router := gin.Default()

	store := cookie.NewStore([]byte(os.Getenv("COOKIE_STORE")))
	store.Options(sessions.Options{
		MaxAge: 3600,
		Path: "/",
		HttpOnly: true,
	})

	router.Use(sessions.Sessions("auth_session", store))

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
		// Create a session
		auth.POST("/login", LoginHandler(db))

		// Delete a session
		auth.GET("/logout", AuthRequired(db), LogoutHandler(db))

		// Create a user
		auth.POST("/register", RegisterHandler(db, v))
	}

	locations := router.Group("/locations")
	locations.Use(AuthRequired(db))
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
	items.Use(AuthRequired(db))
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
	receipts.Use(AuthRequired(db))
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

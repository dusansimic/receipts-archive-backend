package main

import (
	"database/sql"
	"net/http"
	"os"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gbrlsnchs/jwt/v3"
	"github.com/gin-gonic/gin"
	"github.com/jkomyno/nanoid"
	"github.com/jmoiron/sqlx"
	"github.com/markbates/goth"
)

var hs = jwt.NewHS256([]byte(os.Getenv("JWT_KEY")))

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

package main

import (
	"context"
	"log"
	"net/http"
	"os"

	sq "github.com/Masterminds/squirrel"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/jkomyno/nanoid"
	"github.com/jmoiron/sqlx"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
)

// User : Structure that should be used for getting user information from database
type User struct {
	PublicID string `db:"public_id"`
	RealName string `db:"real_name"`
}

type Credentials struct {
	Email string `json:"email" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type StructPublicID struct {
	PublicID string `db:"public_id"`
}

// AuthRequired verifies token sent via request in the cookie and
// checks if the user exists in the database. Afther that adds user id as a
// property inside request context.
func AuthRequired(db *sqlx.DB) gin.HandlerFunc {
	return func (ctx *gin.Context) {
		session := sessions.Default(ctx)

		sessionID := session.Get("session_id")
		if sessionID == nil {
			ctx.String(http.StatusUnauthorized, "Session has expired or is invalid!")
			ctx.Abort()
			return
		}

		userID := session.Get("user_id")
		if userID == nil {
			ctx.String(http.StatusInternalServerError, "Shit just hit the fan while trying to auth!")
			ctx.Abort()
			return
		}

		// See how to extend the session so it won't expire after an hour if used
		// for an hour but will expire if not used for an hour.

		ctx.Set("userID", userID)
		ctx.Next()
	}
}

// CreateSessionID creates a session and returns the id for the user.
func CreateSessionID(ctx *gin.Context, user StructPublicID) error {
	session := sessions.Default(ctx)

	uuid, err := nanoid.Nanoid()
	if err != nil {
		return err
	}

	session.Set("session_id", uuid)
	session.Set("user_id", user.PublicID)
	if err := session.Save(); err != nil {
		return err
	}

	return nil
}

// UserCheck checks if there is a specified user in the database. If there is,
// does nothing. If there is not, inserts the data in database.
func UserCheck(user goth.User, db *sqlx.DB) StructPublicID {
	query := sq.Select("public_id").From("users").Where(sq.Eq{"public_id": user.UserID})
	queryString, queryStringArgs, err := query.ToSql()
	if err != nil {
		log.Fatalln(err.Error())
	}

	userID := StructPublicID{}
	if err := db.Get(&userID, queryString, queryStringArgs...); err != nil {
		insertQuery := sq.Insert("users").Columns("public_id", "real_name").Values(user.UserID, user.Email)
		insertQueryString, insertArgs, err := insertQuery.ToSql()
		if err != nil {
			log.Fatalln(err.Error())
		}

		tx, err := db.Begin()
		if err != nil {
			log.Fatalln(err.Error())
		}

		if _, err := tx.Exec(insertQueryString, insertArgs...); err != nil {
			log.Fatalln(err.Error())
		}

		if err := tx.Commit(); err != nil {
			log.Fatalln(err.Error())
		}

		userID.PublicID = user.UserID
	}

	return userID
}

func AuthHandler(db *sqlx.DB) gin.HandlerFunc {
	return func (ctx *gin.Context) {
		tmpContext := context.WithValue(ctx.Request.Context(), "provider", "google")
		newRequestContext := ctx.Request.WithContext(tmpContext)
		user, err := gothic.CompleteUserAuth(ctx.Writer, newRequestContext)
		if err != nil {
			gothic.BeginAuthHandler(ctx.Writer, newRequestContext)
			return
		}

		userID := UserCheck(user, db)

		if err := CreateSessionID(ctx, userID); err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		ctx.JSON(http.StatusOK, userID)
	}
}

func AuthCallbackHandler(db *sqlx.DB) gin.HandlerFunc {
	return func (ctx *gin.Context) {
		tmpContext := context.WithValue(ctx.Request.Context(), "provider", "google")
		newRequestContext := ctx.Request.WithContext(tmpContext)
		user, err := gothic.CompleteUserAuth(ctx.Writer, newRequestContext)
		if err != nil {
		}

		userID := UserCheck(user, db)

		if err := CreateSessionID(ctx, userID); err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		// Found = MovedTemporarily
		ctx.Redirect(http.StatusFound, os.Getenv("AUTH_CALLBACK"))
	}
}

func LogoutHandler(db *sqlx.DB) gin.HandlerFunc {
	return func (ctx *gin.Context) {
		session := sessions.Default(ctx)

		session.Clear()
		if err := session.Save(); err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		ctx.Status(http.StatusOK)
	}
}

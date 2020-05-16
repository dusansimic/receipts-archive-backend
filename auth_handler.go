package main

import (
	"crypto/sha512"
	"encoding/hex"
	"net/http"

	sq "github.com/Masterminds/squirrel"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator"
	"github.com/jkomyno/nanoid"
	"github.com/jmoiron/sqlx"
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

type StructExtendedID struct {
	*StructID
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
func CreateSessionID(ctx *gin.Context, user StructExtendedID) error {
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

func LoginHandler(db *sqlx.DB) gin.HandlerFunc {
	return func (ctx *gin.Context) {
		var authData Credentials
		if err := ctx.ShouldBindJSON(&authData); err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}

		hasher := sha512.New()
		hasher.Write([]byte(authData.Password))
		passwordHash := hex.EncodeToString(hasher.Sum(nil))

		query := sq.Select("id", "public_id").From("users").Where(sq.Eq{
			"email": authData.Email,
			"password_hash": passwordHash,
		})

		queryString, queryStringArgs, err := query.ToSql()
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		var user StructExtendedID
		if err := db.Get(&user, queryString, queryStringArgs...); err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		if err := CreateSessionID(ctx, user); err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		ctx.Status(http.StatusOK)
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

func RegisterHandler(db *sqlx.DB, v *validator.Validate) gin.HandlerFunc {
	return func (ctx *gin.Context) {
		var authData Credentials
		if err := ctx.ShouldBindJSON(&authData); err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}

		err := v.Struct(authData)
		if err != nil {
			ctx.String(http.StatusBadRequest, err.Error())
			return
		}

		hasher := sha512.New()
		hasher.Write([]byte(authData.Password))
		passwordHash := hex.EncodeToString(hasher.Sum(nil))

		uuid, err := nanoid.Nanoid()
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		query := sq.Insert("users").Columns("public_id", "email", "password_hash").Values(uuid, authData.Email, passwordHash)

		queryString, queryStringArgs, err := query.ToSql()
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		tx, err := db.Begin()
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		if _, err := tx.Exec(queryString, queryStringArgs...); err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		if err := tx.Commit(); err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		ctx.Status(http.StatusOK)
	}
}

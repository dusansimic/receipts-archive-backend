package main

import (
	"context"
	"log"
	"net/http"
	"os"

	sq "github.com/Masterminds/squirrel"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
)

// User : Structure that should be used for getting user information from database
type User struct {
	PublicID string `db:"public_id"`
	RealName string `db:"real_name"`
}

// UserCheck checks if there is a specified user in the database. If there is,
// does nothing. If there is not, inserts the data in database.
func UserCheck(user goth.User, db *sqlx.DB) {
	query := sq.Select("public_id").From("users").Where(sq.Eq{"public_id": user.UserID})
	queryString, queryStringArgs, _ := query.ToSql()

	users := []User{}
	if err := db.Select(&users, queryString, queryStringArgs...); err != nil {
		log.Fatalln(err.Error())
	}

	if len(users) == 0 {
		insertQuery := sq.Insert("users").Columns("public_id", "real_name").Values(user.UserID, user.Email)
		insertQueryString, insertArgs, _ := insertQuery.ToSql()

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
	}
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

		UserCheck(user, db)

		ctx.JSON(http.StatusOK, user)
	}
}

func AuthCallbackHandler(db *sqlx.DB) gin.HandlerFunc {
	return func (ctx *gin.Context) {
		tmpContext := context.WithValue(ctx.Request.Context(), "provider", "google")
		newRequestContext := ctx.Request.WithContext(tmpContext)
		user, err := gothic.CompleteUserAuth(ctx.Writer, newRequestContext)
		if err != nil {
		}

		UserCheck(user, db)

		token, err := CreateToken(user)
		if err != nil {
			ctx.String(http.StatusInternalServerError, err.Error())
			return
		}

		ctx.SetCookie("token", token, 3600, "/", "localhost", false, true)

		ctx.Redirect(http.StatusMovedPermanently, os.Getenv("AUTH_CALLBACK"))
	}
}

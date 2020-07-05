package handlers

import (
	"context"
	"log"
	"net/http"
	"os"

	sq "github.com/Masterminds/squirrel"
	"github.com/bradfitz/gomemcache/memcache"
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

// StructPublicID is a struct for storing only public id
type StructPublicID struct {
	PublicID string `db:"public_id"`
}

type key string

const (
	userIDContextKey = key("userID")
)

// PrivateID gets the database entry id of a user from database that
// corresponds to a specific public id.
func (s *StructPublicID) PrivateID(db *sqlx.DB) (StructID, error) {
	userIDQuery := sq.Select("id").From("users").Where(sq.Eq{"public_id": s.PublicID})
	userIDQueryString, userIDQueryStringArgs, err := userIDQuery.ToSql()
	if err != nil {
		return StructID{}, err
	}

	user := StructID{}
	if err := db.Get(&user, userIDQueryString, userIDQueryStringArgs...); err != nil {
		log.Fatalln(err.Error())
	}

	return user, nil
}

// GetUserID get the user id from specified context. It's literally used just
// so I can write one line instead of two.
func GetUserID(ctx *gin.Context) (StructPublicID, bool) {
	userID, userIDExists := ctx.Get("userID")
	return StructPublicID{
		PublicID: userID.(string),
	}, userIDExists
}

// AuthRequired verifies token sent via request in the cookie and
// checks if the user exists in the database. Afther that adds user id as a
// property inside request context.
func (o Options) AuthRequired() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		session := sessions.Default(ctx)

		sessionID := session.Get("session_id")
		if sessionID == nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"message": "session has expired or is invalid",
			})
			ctx.Abort()
			return
		}
		item, err := o.SessionStore.Get(sessionID.(string))
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"message": "session has expired or is invalid",
			})
			ctx.Abort()
			return
		}

		userID := session.Get("user_id")
		if userID == nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"message": "shit just hit the fan while trying to auth",
			})
			ctx.Abort()
			return
		}

		if userID != string(item.Value) {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"message": "session is invalid",
			})
			ctx.Abort()
			return
		}

		// See how to extend the session so it won't expire after an hour if used
		// for an hour but will expire if not used for an hour.

		// Passing userID inside http request context since GraphQL resolver can only read that one and not the gin context.
		ctx.Request = ctx.Request.WithContext(context.WithValue(ctx.Request.Context(), userIDContextKey, userID))

		ctx.Set("userID", userID)
		ctx.Next()
	}
}

// CreateSessionID creates a session and returns the id for the user.
func (o Options) CreateSessionID(ctx *gin.Context, user StructPublicID) error {
	session := sessions.Default(ctx)

	uuid, err := nanoid.Nanoid()
	if err != nil {
		return err
	}

	sessionStoreItem := &memcache.Item{
		Key:        uuid,
		Value:      []byte(user.PublicID),
		Expiration: 60 * 60, // In seconds (60 seconds is one minute and 60 minutes is one hour)
	}
	if err := o.SessionStore.Set(sessionStoreItem); err != nil {
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
func UserCheck(user goth.User, db *sqlx.DB) (StructPublicID, error) {
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
			return userID, err
		}

		tx, err := db.Begin()
		if err != nil {
			return userID, err
		}

		if _, err := tx.Exec(insertQueryString, insertArgs...); err != nil {
			return userID, err
		}

		if err := tx.Commit(); err != nil {
			return userID, err
		}

		userID.PublicID = user.UserID
	}

	return userID, nil
}

// AuthHandler is Google OAuth handler
func (o Options) AuthHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tmpContext := context.WithValue(ctx.Request.Context(), gothic.ProviderParamKey, "google")
		newRequestContext := ctx.Request.WithContext(tmpContext)
		user, err := gothic.CompleteUserAuth(ctx.Writer, newRequestContext)
		if err != nil {
			gothic.BeginAuthHandler(ctx.Writer, newRequestContext)
			return
		}

		userID, err := UserCheck(user, o.DB)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}

		if err := o.CreateSessionID(ctx, userID); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}

		ctx.JSON(http.StatusOK, userID)
	}
}

// AuthCallbackHandler is Google OAuth callback handler
func (o Options) AuthCallbackHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tmpContext := context.WithValue(ctx.Request.Context(), gothic.ProviderParamKey, "google")
		newRequestContext := ctx.Request.WithContext(tmpContext)
		user, err := gothic.CompleteUserAuth(ctx.Writer, newRequestContext)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}

		userID, err := UserCheck(user, o.DB)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}

		if err := o.CreateSessionID(ctx, userID); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}

		// Found = MovedTemporarily
		ctx.Redirect(http.StatusFound, os.Getenv("AUTH_CALLBACK"))
	}
}

// LogoutHandler is a handler for clearing login session storage
func (o Options) LogoutHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		session := sessions.Default(ctx)

		if err := o.SessionStore.Delete(session.Get("userID").(string)); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}

		session.Clear()
		if err := session.Save(); err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"message": err.Error(),
			})
			return
		}

		ctx.Status(http.StatusOK)
	}
}

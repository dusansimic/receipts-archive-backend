package engine

import (
	"fmt"
	"os"
	"strings"

	"github.com/dusansimic/receipts-archive-backend/handlers"
	"github.com/dusansimic/receipts-archive-backend/handlers/resolvers"
	"github.com/dusansimic/receipts-archive-backend/handlers/stores"
	"github.com/friendsofgo/graphiql"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator"
	"github.com/jmoiron/sqlx"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/google"
)

// NewEngine creates a new Gin engine
func NewEngine(db *sqlx.DB) *gin.Engine {
	router := gin.Default()

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = strings.Split(os.Getenv("ALLOW_ORIGINS"), ",")
	corsConfig.AllowCredentials = true
	router.Use(cors.New(corsConfig))

	session := stores.Session{
		SessionOptions: sessions.Options{
			MaxAge: 3600,
			Path: "/",
			HttpOnly: true,
		},
		Secret: []byte(os.Getenv("SESSION_COOKIE_SECRET")),
	}
	router.Use(sessions.Sessions("auth_session", session.NewSessionStore()))

	// Setup OAuth provider (Google)
	gothic.Store = cookie.NewStore([]byte(os.Getenv("GOTHIC_COOKIE_SECRET")))
	goth.UseProviders(google.New(os.Getenv("GOOGLE_OAUTH_CLIENT_KEY"), os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"), os.Getenv("GOOGLE_OAUTH_CALLBACK_URL")))

	// Request data validator
	v := validator.New()

	// If in debug mode, enable GraphiQL
	if gin.Mode() == gin.DebugMode {
		graphiqlHandler, err := graphiql.NewGraphiqlHandler("/graphql")
		if err != nil {
			fmt.Println(err)
			panic(err)
		}
		router.GET("/graphiql", gin.WrapH(graphiqlHandler))
	}

	auth := router.Group("/auth")
	{
		// Google auth handlers
		// They also create session ids for user
		auth.GET("", handlers.AuthHandler(db))
		auth.GET("/callback", handlers.AuthCallbackHandler(db))

		// Delete a session (logout)
		auth.GET("/logout", handlers.AuthRequired(), handlers.LogoutHandler())
	}

	graphql := router.Group("/graphql")
	{
		// GraphQL request handler
		graphql.POST("", resolvers.GraphQLHandler(db))
	}

	locations := router.Group("/locations")
	locations.Use(handlers.AuthRequired())
	{
		// Get list of locations (query available)
		locations.GET("", handlers.GetLocations(db))

		// Add new location
		locations.POST("", handlers.PostLocations(db))

		// Update location
		locations.PUT("", handlers.PutLocations(db, v))

		// Delete location
		locations.DELETE("", handlers.DeleteLocations(db, v))
	}

	items := router.Group("/items")
	items.Use(handlers.AuthRequired())
	{
		// Get list of items (query available)
		items.GET("", handlers.GetItems(db))

		// Get list of items from a specific receipt
		items.GET("/inreceipt/:id", handlers.GetItemsInReceipt(db))

		// Add new item
		items.POST("", handlers.PostItems(db, v))

		// Add item to receipts
		items.POST("/inreceipt", handlers.PostItemsInReceipt(db, v))

		// Update item
		items.PUT("", handlers.PutItems(db, v))

		// Update item from specific receipt
		items.PUT("/inreceipt", handlers.PutItemsInReceipt(db))

		// Delete item
		items.DELETE("", handlers.DeleteItems(db, v))

		// Delete item from receipt
		items.DELETE("/inreceipt", handlers.DeleteItemsInReceipt(db, v))
	}

	receipts := router.Group("/receipts")
	receipts.Use(handlers.AuthRequired())
	{
		// Get list of receipts (query available)
		receipts.GET("", handlers.GetReceipts(db))

		// Add new receipt
		receipts.POST("", handlers.PostReceipts(db, v))

		// Update receipt
		receipts.PUT("", handlers.PutReceipts(db, v))

		// Delete receipt
		receipts.DELETE("", handlers.DeleteReceipts(db, v))
	}

	return router
}

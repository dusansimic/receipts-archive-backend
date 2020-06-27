package engine

import (
	"fmt"

	"github.com/dusansimic/receipts-archive-backend/handlers"
	"github.com/dusansimic/receipts-archive-backend/handlers/resolvers"
	"github.com/dusansimic/receipts-archive-backend/handlers/stores"
	"github.com/friendsofgo/graphiql"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator"
	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/google"
)

// GoogleOAuthOptions stores options for Google OAuth
type GoogleOAuthOptions struct {
	ClientKey    string
	ClientSecret string
	CallbackURL  string
}

// Options stores options for new engine
type Options struct {
	AllowOrigins        []string
	SessionCookieSecret []byte
	GothicCookieSecret  []byte
	GoogleOAuthOptions
}

// NewEngine creates a new Gin engine
func (o Options) NewEngine(db *sqlx.DB, rdb *redis.Client) *gin.Engine {
	router := gin.Default()

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = o.AllowOrigins
	corsConfig.AllowCredentials = true
	router.Use(cors.New(corsConfig))

	session := stores.Session{
		SessionOptions: sessions.Options{
			MaxAge:   3600,
			Path:     "/",
			HttpOnly: true,
		},
		Secret: o.SessionCookieSecret,
	}
	router.Use(sessions.Sessions("auth_session", session.NewSessionStore()))

	// Setup OAuth provider (Google)
	gothic.Store = cookie.NewStore(o.GothicCookieSecret)
	goth.UseProviders(google.New(o.GoogleOAuthOptions.ClientKey, o.GoogleOAuthOptions.ClientSecret, o.GoogleOAuthOptions.CallbackURL))

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

	handlers := handlers.Options{
		DB:  db,
		RDB: rdb,
		V:   v,
	}
	resolvers := resolvers.Options{
		DB: db,
	}

	auth := router.Group("/auth")
	{
		// Google auth handlers
		// They also create session ids for user
		auth.GET("", handlers.AuthHandler())
		auth.GET("/callback", handlers.AuthCallbackHandler())

		// Delete a session (logout)
		auth.GET("/logout", handlers.AuthRequired(), handlers.LogoutHandler())
	}

	graphql := router.Group("/graphql")
	{
		// GraphQL request handler
		graphql.POST("", resolvers.GraphQLHandler())
	}

	locations := router.Group("/locations")
	locations.Use(handlers.AuthRequired())
	{
		// Get list of locations (query available)
		locations.GET("", handlers.GetLocations())

		// Add new location
		locations.POST("", handlers.PostLocations())

		// Update location
		locations.PUT("", handlers.PutLocations())

		// Delete location
		locations.DELETE("", handlers.DeleteLocations())
	}

	items := router.Group("/items")
	items.Use(handlers.AuthRequired())
	{
		// Get list of items (query available)
		items.GET("", handlers.GetItems())

		// Get list of items from a specific receipt
		items.GET("/inreceipt/:id", handlers.GetItemsInReceipt())

		// Add new item
		items.POST("", handlers.PostItems())

		// Add item to receipts
		items.POST("/inreceipt", handlers.PostItemsInReceipt())

		// Update item
		items.PUT("", handlers.PutItems())

		// Update item from specific receipt
		items.PUT("/inreceipt", handlers.PutItemsInReceipt())

		// Delete item
		items.DELETE("", handlers.DeleteItems())

		// Delete item from receipt
		items.DELETE("/inreceipt", handlers.DeleteItemsInReceipt())
	}

	receipts := router.Group("/receipts")
	receipts.Use(handlers.AuthRequired())
	{
		// Get list of receipts (query available)
		receipts.GET("", handlers.GetReceipts())

		// Add new receipt
		receipts.POST("", handlers.PostReceipts())

		// Update receipt
		receipts.PUT("", handlers.PutReceipts())

		// Delete receipt
		receipts.DELETE("", handlers.DeleteReceipts())
	}

	return router
}

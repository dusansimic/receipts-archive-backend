package main

import (
	"fmt"
	"os"
	"strings"

	// Server related stuff
	"github.com/dusansimic/receipts-archive-backend/database"
	"github.com/dusansimic/receipts-archive-backend/engine"

	// Other stuff
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	// Setup database
	sqlDB := database.SQLOptions{
		DatabasePath: "./receipts.db",
	}
	db, err := sqlDB.GenerateDatabase()
	if err != nil {
		fmt.Println("Failed to connect to the database!")
		fmt.Println(err)
		panic(err)
	}

	redisDB := database.RedisOptions{
		Addr:     os.Getenv("SESSION_CACHE_DATABASE_ADDRESS"),
		Password: os.Getenv("SESSION_CACHE_DATABASE_PASSWORD"),
		DB:       0,
	}
	rdb := redisDB.NewConnection()

	// engn because engine is used
	engn := engine.Options{
		AllowOrigins:        strings.Split(os.Getenv("ALLOW_ORIGINS"), ","),
		SessionCookieSecret: []byte(os.Getenv("SESSION_COOKIE_SECRET")),
		GothicCookieSecret:  []byte(os.Getenv("GOTHIC_COOKIE_SECRET")),
		GoogleOAuthOptions: engine.GoogleOAuthOptions{
			ClientKey:    os.Getenv("GOOGLE_OAUTH_CLIENT_KEY"),
			ClientSecret: os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"),
			CallbackURL:  os.Getenv("GOOGLE_OAUTH_CALLBACK_URL"),
		},
	}
	router := engn.NewEngine(db, rdb)

	router.Run(":" + os.Getenv("PORT"))
}

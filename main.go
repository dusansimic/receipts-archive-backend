package main

import (
	"fmt"
	"os"

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
		Addr: os.Getenv("SESSION_CACHE_DATABASE_ADDRESS"),
		Password: os.Getenv("SESSION_CACHE_DATABASE_PASSWORD"),
		DB: 0,
	}
	rdb := redisDB.NewConnection()

	router := engine.NewEngine(db, rdb)

	router.Run(":" + os.Getenv("PORT"))
}

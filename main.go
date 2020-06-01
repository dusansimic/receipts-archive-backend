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
	database := database.Options{
		DatabasePath: "./receipts.db",
	}
	db, err := database.GenerateDatabase()
	if err != nil {
		fmt.Println("Failed to connect to the database!")
		fmt.Println(err)
		panic(err)
	}

	router := engine.NewEngine(db)

	router.Run(":" + os.Getenv("PORT"))
}

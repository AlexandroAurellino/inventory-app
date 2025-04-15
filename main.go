// main.go
package main

import (
	"inventory-app/config"
	"inventory-app/routes"
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize the database.
	config.InitDB()

	// Create a new Gin router.
	router := gin.Default()

	// Register the API routes.
	routes.RegisterRoutes(router)

	// Optionally, set the port from an environment variable.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on port %s...", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}

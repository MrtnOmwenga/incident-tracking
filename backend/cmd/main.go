package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/joho/godotenv"

	"github.com/MrtnOmwenga/incident-tracking/backend/pkg/config"
	container "github.com/MrtnOmwenga/incident-tracking/backend/pkg/containers"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	PORT := os.Getenv("PORT")

	db := config.GetDB()
	app, err := container.Init(db)
	if err != nil {
		log.Fatalf("Failed to initialize app container: %v", err)
	}

	router := app.GetRouter()
	corsMiddleware := handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}), // Replace with your frontend's URL
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
	)

	log.Printf("Server started on %s", PORT)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", PORT), corsMiddleware(router)))
}

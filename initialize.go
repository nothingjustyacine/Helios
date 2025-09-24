package main

import (
	"log"
	"os"

	"helios/config"
	"helios/database"
)

func initAll() {
	subscriptionURL := os.Getenv("SUBSCRIPTION_URL")
	if subscriptionURL == "" {
		log.Fatal("Error: SUBSCRIPTION_URL environment variable is not set")
	}

	log.Println("Fetching subscription configuration...")
	if err := config.FetchSubscription(subscriptionURL); err != nil {
		log.Fatalf("Failed to fetch subscription: %v", err)
	}

	log.Println("Initializing database...")
	dbPath := "/data/helios.db"
	// dbPath := "helios.db"
	if err := database.InitializeDatabase(dbPath); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	log.Println("All initialization completed successfully")
}

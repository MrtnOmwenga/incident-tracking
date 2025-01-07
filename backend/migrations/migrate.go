package main

import (
    "database/sql"
    "fmt"
    "log"
    "os"
    "path/filepath"

    _ "github.com/lib/pq"
    "github.com/golang-migrate/migrate/v4"
    "github.com/golang-migrate/migrate/v4/database/postgres"
    _ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
    // Construct database connection string
    dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
        os.Getenv("DB_USER"),
        os.Getenv("DB_PASSWORD"),
        os.Getenv("DB_HOST"),
        os.Getenv("DB_PORT"),
        os.Getenv("DB_NAME"),
        os.Getenv("DB_SSL_MODE"),
    )

    // Connect to database
    db, err := sql.Open("postgres", dbURL)
    if err != nil {
			log.Fatalf("Could not connect to the database: %v", err)
    }
    defer db.Close()

    // Test the connection
    if err = db.Ping(); err != nil {
			log.Fatalf("Could not ping the database: %v", err)
    }

    // Create postgres driver instance
    driver, err := postgres.WithInstance(db, &postgres.Config{})
    if err != nil {
			log.Fatalf("Could not create the postgres driver: %v", err)
    }

    // Get the absolute path to migrations
    migrationsPath := filepath.Join("file:///app/migrations")

    // Create new migrate instance
    m, err := migrate.NewWithDatabaseInstance(
			migrationsPath,
			"postgres", 
			driver,
    )
    if err != nil {
			log.Fatalf("Migration failed: %v", err)
    }

    // Run migrations up
    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("An error occurred while executing migrations: %v", err)
    }

    log.Println("Migrations completed successfully")
}
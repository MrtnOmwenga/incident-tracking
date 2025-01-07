package config

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/MrtnOmwenga/incident-tracking/backend/pkg/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	db   *gorm.DB
	once sync.Once
)

func GetDB() *gorm.DB {
	once.Do(func() {
		host := os.Getenv("DB_HOST")
		port := os.Getenv("DB_PORT")
		user := os.Getenv("DB_USER")
		password := os.Getenv("DB_PASSWORD")
		dbname := os.Getenv("DB_NAME")

		// Connection string
		dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			host, port, user, password, dbname)

		// Initialize the database connection
		var err error
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
		log.Println("Successfully connected to PostgreSQL!")

		err = db.AutoMigrate(
			&models.Incident{},
			&models.Comment{},
			&models.Image{},
			&models.CommentImage{},
		)
		if err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}
		log.Println("Database migrated successfully!")
	})

	return db
}

package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

// envOrDefault returns the env var value or the given fallback.
func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// connectDB opens a PostgreSQL connection using env vars and verifies it.
func connectDB() (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		envOrDefault("POSTGRES_HOST", "localhost"),
		envOrDefault("POSTGRES_PORT", "5432"),
		envOrDefault("POSTGRES_USER", "postgres"),
		envOrDefault("POSTGRES_PASSWORD", "postgres"),
		envOrDefault("POSTGRES_NAME", "k8learn"),
		envOrDefault("POSTGRES_SSLMODE", "disable"),
	)
	fmt.Printf("connected DB %s:%s\n",envOrDefault("POSTGRES_HOST", "localhost"),envOrDefault("POSTGRES_PORT", "5432"))
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("db.Ping: %w", err)
	}

	log.Println("Connected to PostgreSQL")
	return db, nil
}

// migrate creates the students table if it doesn't already exist.
func migrate(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS students (
		id   SERIAL PRIMARY KEY,
		name TEXT   NOT NULL,
		age  INT    NOT NULL
	);`

	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	log.Println("Database migrated (students table ready)")
	return nil
}

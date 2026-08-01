package main

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"os"
)

//go:embed templates/*.html static/*
var embeddedFS embed.FS

func main() {
	// --- Database ---
	db, err := connectDB()
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer db.Close()

	if err := migrate(db); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	r := &repo{db: db}

	// --- Templates ---
	tmpl := template.Must(template.New("").ParseFS(embeddedFS, "templates/*.html"))

	// --- Router ---
	mux := http.NewServeMux()

	// Static files
	mux.Handle("GET /static/", http.FileServer(http.FS(embeddedFS)))

	// Page & action routes
	mux.HandleFunc("GET /", withError(indexHandler(tmpl, r)))
	mux.HandleFunc("GET /students/new", withError(newHandler(tmpl)))
	mux.HandleFunc("POST /students", withError(createHandler(tmpl, r)))
	mux.HandleFunc("GET /students/{id}/edit", withError(editHandler(tmpl, r)))
	mux.HandleFunc("POST /students/{id}/update", withError(updateHandler(tmpl, r)))
	mux.HandleFunc("POST /students/{id}/delete", withError(deleteHandler(tmpl, r)))

	// --- Start ---
	port := envOrDefault("PORT", "8080")
	log.Printf("Server listening on http://0.0.0.0:%s", port)
	if err := http.ListenAndServe("0.0.0.0:"+port, mux); err != nil {
		log.Fatalf("Server error: %v", err)
		os.Exit(1)
	}
}

package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
)

// repo wraps database access for students.
type repo struct {
	db *sql.DB
}

// listAll returns every student ordered by id.
func (r *repo) listAll() ([]Student, error) {
	rows, err := r.db.Query("SELECT id, name, age FROM students ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var students []Student
	for rows.Next() {
		var s Student
		if err := rows.Scan(&s.ID, &s.Name, &s.Age); err != nil {
			return nil, err
		}
		students = append(students, s)
	}
	return students, rows.Err()
}

// getByID returns a single student or sql.ErrNoRows.
func (r *repo) getByID(id int) (Student, error) {
	var s Student
	err := r.db.QueryRow("SELECT id, name, age FROM students WHERE id = $1", id).
		Scan(&s.ID, &s.Name, &s.Age)
	return s, err
}

// insert creates a student and returns the new record with its generated id.
func (r *repo) insert(name string, age int) (Student, error) {
	var s Student
	err := r.db.QueryRow(
		"INSERT INTO students (name, age) VALUES ($1, $2) RETURNING id, name, age",
		name, age,
	).Scan(&s.ID, &s.Name, &s.Age)
	return s, err
}

// update modifies an existing student. Returns the updated row.
func (r *repo) update(id int, name string, age int) (Student, error) {
	var s Student
	err := r.db.QueryRow(
		"UPDATE students SET name = $1, age = $2 WHERE id = $3 RETURNING id, name, age",
		name, age, id,
	).Scan(&s.ID, &s.Name, &s.Age)
	return s, err
}

// delete removes a student by id.
func (r *repo) delete(id int) error {
	res, err := r.db.Exec("DELETE FROM students WHERE id = $1", id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("student %d not found", id)
	}
	return nil
}

// parseID extracts an integer id from a URL path value.
func parseID(s string) (int, error) {
	return strconv.Atoi(s)
}

// parseForm extracts name and age from a POST form. Simple validation.
func parseForm(r *http.Request) (name string, age int, err error) {
	name = r.FormValue("name")
	if name == "" {
		return "", 0, fmt.Errorf("name is required")
	}
	ageStr := r.FormValue("age")
	age, err = strconv.Atoi(ageStr)
	if err != nil || age <= 0 {
		return "", 0, fmt.Errorf("age must be a positive integer")
	}
	return name, age, nil
}

// h is a convenience type for handler functions that return an error.
type h func(w http.ResponseWriter, r *http.Request) error

// withError wraps an h so that returned errors become 500 responses.
func withError(fn h) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			log.Printf("ERROR %s %s: %v", r.Method, r.URL.Path, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

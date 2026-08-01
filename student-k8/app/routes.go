package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
)

// ---- Page handlers ----

func indexHandler(tmpl *template.Template, r *repo) h {
	return func(w http.ResponseWriter, req *http.Request) error {
		students, err := r.listAll()
		if err != nil {
			return fmt.Errorf("list students: %w", err)
		}
		return tmpl.ExecuteTemplate(w, "base.html", tmplData{
			Title:    "Students",
			Students: students,
		})
	}
}

func newHandler(tmpl *template.Template) h {
	return func(w http.ResponseWriter, req *http.Request) error {
		return tmpl.ExecuteTemplate(w, "base.html", tmplData{
			Title: "Add Student",
			Form:  &formData{},
		})
	}
}

func editHandler(tmpl *template.Template, r *repo) h {
	return func(w http.ResponseWriter, req *http.Request) error {
		id, err := parseID(req.PathValue("id"))
		if err != nil {
			return fmt.Errorf("invalid id: %w", err)
		}
		s, err := r.getByID(id)
		if err != nil {
			if err == sql.ErrNoRows {
				http.NotFound(w, req)
				return nil
			}
			return fmt.Errorf("get student %d: %w", id, err)
		}
		return tmpl.ExecuteTemplate(w, "base.html", tmplData{
			Title:    "Edit Student",
			Students: []Student{s},
			Form:     &formData{Name: s.Name, Age: s.Age},
			Editing:  true,
			EditID:   s.ID,
		})
	}
}

// ---- Action handlers ----

func createHandler(tmpl *template.Template, r *repo) h {
	return func(w http.ResponseWriter, req *http.Request) error {
		name, age, verr := parseForm(req)
		if verr != nil {
			return tmpl.ExecuteTemplate(w, "base.html", tmplData{
				Title: "Add Student",
				Form:  &formData{Name: name, Age: age},
				Error: verr.Error(),
			})
		}
		_, err := r.insert(name, age)
		if err != nil {
			return fmt.Errorf("insert student: %w", err)
		}
		http.Redirect(w, req, "/", http.StatusSeeOther)
		return nil
	}
}

func updateHandler(tmpl *template.Template, r *repo) h {
	return func(w http.ResponseWriter, req *http.Request) error {
		id, err := parseID(req.PathValue("id"))
		if err != nil {
			return fmt.Errorf("invalid id: %w", err)
		}
		name, age, verr := parseForm(req)
		if verr != nil {
			s, _ := r.getByID(id)
			return tmpl.ExecuteTemplate(w, "base.html", tmplData{
				Title:    "Edit Student",
				Students: []Student{s},
				Form:     &formData{Name: name, Age: age},
				Editing:  true,
				EditID:   id,
				Error:    verr.Error(),
			})
		}
		if _, err := r.update(id, name, age); err != nil {
			return fmt.Errorf("update student %d: %w", id, err)
		}
		http.Redirect(w, req, "/", http.StatusSeeOther)
		return nil
	}
}

func deleteHandler(tmpl *template.Template, r *repo) h {
	return func(w http.ResponseWriter, req *http.Request) error {
		id, err := parseID(req.PathValue("id"))
		if err != nil {
			return fmt.Errorf("invalid id: %w", err)
		}
		if err := r.delete(id); err != nil {
			return fmt.Errorf("delete student %d: %w", id, err)
		}
		http.Redirect(w, req, "/", http.StatusSeeOther)
		return nil
	}
}

// ---- Template data types ----

type tmplData struct {
	Title    string
	Students []Student
	Form     *formData
	Error    string
	Editing  bool   // true when showing edit form
	EditID   int    // id of the student being edited
}

type formData struct {
	Name string
	Age  int
}

package main

import (
	"log"
	"net/http"
	"text/template"
)

func BaseHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("./cmd/log_viewer/views/_base.html"))

	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func ProgramsHandler(w http.ResponseWriter, r *http.Request) {
	db, err := connectDB()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	programs, err := getPrograms(db)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Parse both base and programs templates
	tmpl, err := template.ParseFiles(
		"./cmd/log_viewer/views/_base.html",
		"./cmd/log_viewer/views/programs.html",
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Println(err.Error())
		return
	}

	// Execute the base template with the "content" block from programs template
	if err := tmpl.ExecuteTemplate(w, "content", programs); err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

package main

import (
	"fmt"
	"net/http"
	"text/template"
)

//var tmpl *template.Template

func BaseHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("./cmd/log_viewer/views/_base.html"))

	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func ProgramsHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("ProgramsHandler")
	db, err := connectDB()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	defer db.Close()
	fmt.Println("Connected to DB")

	fmt.Print("Getting SQL items")
	programs, err := getPrograms(db)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	fmt.Printf("%+v", programs)

	tmpl := template.Must(template.ParseFiles("./cmd/log_viewer/views/programs.html"))

	if err := tmpl.Execute(w, programs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

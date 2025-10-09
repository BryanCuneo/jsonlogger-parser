package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"text/template"
)

var tmpl *template.Template

func sendErr(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")
	resp := make(map[string]string)
	resp["message"] = message
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error happened in JSON marshal. Err: %s\n", err)
		return
	}
	w.Write(jsonResp)
	log.Println(jsonResp)
	return
}

func BaseHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("cmd/log_viewer/views/base_page.html"))
	tmpl.Execute(w, nil)
}

func ProgramsHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("ProgramsHandler")
	db, err := connectDB()
	if err != nil {
		sendErr(w, 500, "Internal Server Error")
	}
	defer db.Close()

	programs, err := getPrograms(db)
	if err != nil {
		sendErr(w, 500, "Internal Server Error")
	}
	fmt.Printf("%+v", programs)

	tmpl := template.Must(template.ParseFiles("cmd/log_viewer/views/programs_list.html"))
	tmpl.Execute(w, programs)
}

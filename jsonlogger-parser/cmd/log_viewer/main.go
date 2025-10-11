package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/microsoft/go-mssqldb"
)

type Program struct {
	ID              int64
	Name            string
	Log_folder_path string
	Archive_days    int16
	Delete_days     int16
}

type Session struct {
	Id           int64
	Program_id   int64
	Has_warning  bool
	Has_error    bool
	Has_fatal    bool
	Created_date time.Time
	Is_archived  bool
}

func connectDB() (*sql.DB, error) {
	db, err := sql.Open("mssql", os.Getenv("SQL_CONN_STRING"))
	if err != nil {
		return nil, err
	}
	return db, nil
}

func getPrograms(db *sql.DB) ([]Program, error) {
	programsQuery := "select * from programs order by program_name"
	rows, err := db.Query(programsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var programs []Program
	for rows.Next() {
		var prog Program
		if err := rows.Scan(
			&prog.ID, &prog.Name, &prog.Log_folder_path,
			&prog.Archive_days, &prog.Delete_days); err != nil {
			return nil, err
		}
		programs = append(programs, prog)
	}

	return programs, nil
}

func getSessions(db *sql.DB, programID string) ([]Session, error) {
	sessionsQuery := "select * from log_sessions where program_id = ? order by created_date desc"
	rows, err := db.Query(sessionsQuery, programID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var sesh Session
		if err := rows.Scan(
			&sesh.Id, &sesh.Program_id, &sesh.Has_warning, &sesh.Has_error,
			&sesh.Has_fatal, &sesh.Created_date, &sesh.Is_archived); err != nil {
			return nil, err
		}
		sessions = append(sessions, sesh)
	}

	return sessions, nil
}

// func getSession(db *sql.DB, programID string) (Session, error) {
// 	sessionsQuery := "select top(1) * from log_sessions where program_id = ?"
// 	rows, err := db.Query(sessionsQuery, programID)
// 	if err != nil {
// 		return Session{}, err
// 	}
// 	defer rows.Close()

// 	var session Session
// 	for rows.Next() {
// 		if err := rows.Scan(
// 			&session.Id, &session.Program_id, &session.Has_warning, &session.Has_error,
// 			&session.Has_fatal, &session.Created_date, &session.Is_archived); err != nil {
// 			return Session{}, err
// 		}
// 	}

// 	// If no session found, return an error
// 	if session.Id == 0 {
// 		return Session{}, err
// 	}

// 	return session, nil
// }

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file: ", err.Error())
	}
}

func main() {
	// Handle static files
	http.Handle("/stylesheets/", http.StripPrefix("/stylesheets/", http.FileServer(http.Dir("cmd/log_viewer/static/styles"))))

	http.HandleFunc("/", BaseHandler)
	http.HandleFunc("/programs", ProgramsHandler)
	http.HandleFunc("/programs/{slug}", SessionsHandler)

	log.Fatal(http.ListenAndServe(":8000", nil))
}

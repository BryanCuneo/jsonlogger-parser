package main

import (
	"database/sql"
	"fmt"
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
	log_folder_path string
	archive_days    int16
	delete_days     int16
}

type Session struct {
	id           int64
	program_id   int64
	has_warning  bool
	has_error    bool
	has_fatal    bool
	created_date time.Time
	is_archived  bool
}

func connectDB() (*sql.DB, error) {
	db, err := sql.Open("mssql", os.Getenv("SQL_CONN_STRING"))
	if err != nil {
		return nil, err
	}
	return db, nil
}

func getPrograms(db *sql.DB) ([]Program, error) {
	programsQuery := "select * from programs"
	rows, err := db.Query(programsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var programs []Program
	for rows.Next() {
		var prog Program
		if err := rows.Scan(
			&prog.ID, &prog.Name, &prog.log_folder_path,
			&prog.archive_days, &prog.delete_days); err != nil {
			return nil, err
		}
		programs = append(programs, prog)
	}

	return programs, nil
}

// func getSessions(db *sql.DB) ([]Session, error) {
// 	sessionsQuery := "select * from log_sessions"
// 	rows, err := db.Query(sessionsQuery)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var sessions []Session
// 	for rows.Next() {
// 		var sesh Session
// 		if err := rows.Scan(
// 			&sesh.id, &sesh.program_id, &sesh.has_warning, &sesh.has_error,
// 			&sesh.has_fatal, &sesh.created_date, &sesh.is_archived); err != nil {
// 			return nil, err
// 		}
// 		sessions = append(sessions, sesh)
// 	}

// 	return sessions, nil
// }

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file: ", err.Error())
	}
}

func main() {

	db, err := connectDB()
	if err != nil {
		log.Println("Error connecting to DB: ", err.Error())
	}
	defer db.Close()

	programs, err := getPrograms(db)
	if err != nil {
		log.Fatal("Unable to connect to DB: ", err.Error())

	}
	fmt.Printf("%+v", programs)

	http.HandleFunc("/", BaseHandler)
	http.HandleFunc("/programs", ProgramsHandler)

	log.Fatal(http.ListenAndServe(":8000", nil))
}

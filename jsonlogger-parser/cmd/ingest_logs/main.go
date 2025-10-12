package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/microsoft/go-mssqldb"
)

type Program struct {
	program_id      int
	program_name    string
	log_folder_path string
}

type PsTimestamp struct {
	time.Time
}

func (ct *PsTimestamp) UnmarshalJSON(b []byte) error {
	str := string(b[1 : len(b)-1]) // Trim the quotes
	layout := "2006-01-02T15:04:05.999999999-07:00"
	parsedTime, err := time.Parse(layout, str)
	if err != nil {
		return err
	}
	ct.Time = parsedTime
	return nil
}

type InitialEntry struct {
	Timestamp         PsTimestamp `json:"timestamp"`
	Level             string      `json:"level"`
	ProgramName       string      `json:"programName"`
	PSVersion         string      `json:"PSVersion"`
	JsonLoggerVersion string      `json:"jsonLoggerVersion"`
	HasWarning        bool        `json:"hasWarning,omitempty"`
	HasError          bool        `json:"hasError,omitempty"`
	HasFatal          bool        `json:"hasFatal,omitempty"`
}

func archiveFile(filePath string) error {
	logFileName := filepath.Base(filePath)
	zipFilename := strings.Replace(logFileName, filepath.Ext(logFileName), ".zip", 1)
	parentFolder := filepath.Base(filepath.Dir(filePath))
	zipDestination := filepath.Join(os.Getenv("ARCHIVE_PATH"), parentFolder, zipFilename)

	err := os.MkdirAll(filepath.Dir(zipDestination), os.ModePerm)
	if err != nil {
		return err
	}

	output, err := os.Create(zipDestination)
	if err != nil {
		return err
	}
	defer output.Close()

	zipWriter := zip.NewWriter(output)
	defer zipWriter.Close()

	input, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer input.Close()

	zipEntry, err := zipWriter.Create(logFileName)
	if err != nil {
		return err
	}

	_, err = io.Copy(zipEntry, input)
	return err
}

func contains(slice []string, item string) bool {
	for _, element := range slice {
		if element == item {
			return true
		}
	}
	return false
}

func getNewFolders(path string, existingFolders []string) ([]string, error) {
	var newFolders []string

	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Check if it's a directory
		if info.IsDir() && !contains(existingFolders, filePath) && filePath != path {
			newFolders = append(newFolders, filePath)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return newFolders, nil
}

func insertProgram(db *sql.DB, path string) error {
	name := filepath.Base(path)
	query := "insert into programs (program_name, log_folder_path) values (?, ?)"
	_, err := db.Exec(query, name, path)

	return err
}

func insertNewPrograms(db *sql.DB, path string) (int, error) {
	var program_paths_in_db []string

	get_programs_query := "select log_folder_path from programs"
	rows, err := db.Query(get_programs_query)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	if err = db.Ping(); err != nil {
		log.Fatal("Database is not reachable: ", err)
	}

	var value string
	for rows.Next() {
		err = rows.Scan(&value)
		if err != nil {
			log.Fatal("Error scanning row: ", err.Error())
		}
		program_paths_in_db = append(program_paths_in_db, value)
	}

	new_paths, err := getNewFolders(path, program_paths_in_db)
	if err != nil {
		return 0, err
	}

	for _, path := range new_paths {
		err := insertProgram(db, path)
		if err != nil {
			fmt.Printf(" ERROR!\n%s\n", err.Error())
		}
	}

	return len(new_paths), nil
}

func insertLogEntries(db *sql.DB, sessionId int, logFilePath string) error {
	updateSessionQuery := "update log_sessions set has_warning = ?, has_error = ?, has_fatal = ? where _id = ?"
	insertLogEntryQuery := "insert into log_entries (session_id, log_entry) values (?, ?)"

	file, err := os.Open(logFilePath)
	if err != nil {
		fmt.Printf("Error opening %s:\n%s\n", logFilePath, err.Error())
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Check for BOM, remove if found, and insert first line
	scanner.Scan()
	firstLine := scanner.Bytes()
	if after, ok := bytes.CutPrefix(firstLine, []byte{0xEF, 0xBB, 0xBF}); ok {
		firstLine = after
	}

	//TODO Update session with hasWarning/hasError/hasFatal
	var initialEntry InitialEntry
	if err := json.Unmarshal(firstLine, &initialEntry); err != nil {
		fmt.Println("Error unmarshalling initial entry")
		return err
	}

	if initialEntry.HasWarning || initialEntry.HasError || initialEntry.HasFatal {
		_, err = db.Exec(updateSessionQuery, initialEntry.HasWarning, initialEntry.HasError, initialEntry.HasFatal, sessionId)
		if err != nil {
			fmt.Println("Error updating session")
			return err
		}
	}

	_, err = db.Exec(insertLogEntryQuery, sessionId, string(firstLine))
	if err != nil {
		fmt.Println("Error inserting initial log entry: ", string(firstLine))
		return err
	}
	// Insert the rest of the log entries
	for scanner.Scan() {
		line := scanner.Text()

		_, err := db.Exec(insertLogEntryQuery, sessionId, line)
		if err != nil {
			fmt.Println("Error inserting log entry: ", line)
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error scanning file: ", logFilePath)
		return err
	}

	return nil
}

func insertNewSession(db *sql.DB, programId int, filePath string) error {
	query := "insert into log_sessions (program_id, created_date) output inserted._id values (?, ?)"
	var createdDate time.Time

	switch {
	case runtime.GOOS == "windows":
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			return err
		}
		winInfo := fileInfo.Sys().(*syscall.Win32FileAttributeData)
		createdDate = time.Unix(0, winInfo.CreationTime.Nanoseconds())
	default:
		createdDate = time.Now()
	}

	var newSessionId int
	err := db.QueryRow(query, programId, createdDate).Scan(&newSessionId)
	if err != nil {
		fmt.Print("Error scanning row: ", err.Error())
		return err
	}

	err = insertLogEntries(db, newSessionId, filePath)
	if err != nil {
		fmt.Println("Error inserting log entry: ", err.Error())
		return err
	}

	return err
}

func insertNewLogs(db *sql.DB) error {
	get_programs_query := "select _id, program_name, log_folder_path from programs"
	programs, err := db.Query(get_programs_query)
	if err != nil {
		fmt.Println("Error selecting from programs: ", err.Error())
		return err
	}
	defer programs.Close()

	var program Program
	for programs.Next() {
		err = programs.Scan(&program.program_id, &program.program_name, &program.log_folder_path)
		if err != nil {
			log.Fatal("Error scanning row: ", err.Error())
		}

		var files []string
		err := filepath.Walk(program.log_folder_path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Printf("Error walking %s:\n%s\n", program.log_folder_path, err.Error())
				return err
			}
			if !info.IsDir() {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			fmt.Println("Error getting log files: ", err.Error())
		}

		fmt.Printf("Inserting %d new logs for %s\n", len(files), program.program_name)

		for _, file := range files {
			err = insertNewSession(db, program.program_id, file)
			if err != nil {
				fmt.Println("Error inserting new session: ", err.Error())
			} else {
				err = archiveFile(file)
				if err != nil {
					fmt.Println("Error archiving file: ", err.Error())
				} else {
					os.Remove(file)
				}
			}
		}
	}

	return err
}

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file: ", err.Error())
	}
}

func main() {
	db, err := sql.Open("mssql", os.Getenv("SQL_CONN_STRING"))
	if err != nil {
		log.Fatal("Error connecting to database: ", err.Error())
	}
	defer db.Close()
	err = db.Ping()
	if err != nil {
		log.Fatal("Database is not reachable: ", err)
	}

	fmt.Print("Checking for new programs...")
	newProgramsCount, err := insertNewPrograms(db, os.Getenv("LOGS_PATH"))
	if err != nil {
		fmt.Println("Error: ", err.Error())
	}
	fmt.Printf(" %d\n\n", newProgramsCount)

	err = insertNewLogs(db)
	if err != nil {
		log.Fatal("Error inserting new logs: ", err.Error())
	}

	fmt.Println("\nAll done")

	db.Close()
}

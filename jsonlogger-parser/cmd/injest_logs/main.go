package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/microsoft/go-mssqldb"
)

type Program struct {
	program_id      int
	program_name    string
	log_folder_path string
}

var dataSource = "localhost\\SQLEXPRESS"
var initialCatalog = "ps_log_store"
var logsRootPath = "C:\\Users\\Bryan Cuneo\\source\\jsonlogger-parser\\ignore\\sample_logs"
var logsArchivePath = "C:\\Users\\Bryan Cuneo\\source\\jsonlogger-parser\\ignore\\archive"

func archiveFile(filePath string) error {
	logFileName := filepath.Base(filePath)
	zipFilename := strings.Replace(logFileName, filepath.Ext(logFileName), ".zip", 1)
	parentFolder := filepath.Base(filepath.Dir(filePath))
	zipDestination := filepath.Join(logsArchivePath, parentFolder, zipFilename)

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

func insertLogEntry(db *sql.DB, sessionId int, logFilePath string) error {
	query := "insert into log_entries (session_id, log_entry) values (?, ?)"

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
	_, err = db.Exec(query, sessionId, string(firstLine))
	if err != nil {
		fmt.Println("Error inserting row: ", err.Error())
		return err
	}
	// Insert the rest of the log entries
	for scanner.Scan() {
		line := scanner.Text()

		_, err := db.Exec(query, sessionId, line)
		if err != nil {
			fmt.Println("Error inserting row: ", err.Error())
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error scanning file: ", err.Error())
		return err
	}

	return nil
}

func insertNewSession(db *sql.DB, programId int, filePath string) error {
	query := "insert into log_sessions (program_id) output inserted._id values (?)"

	var newSessionId int
	err := db.QueryRow(query, programId).Scan(&newSessionId)
	if err != nil {
		fmt.Print("Error scanning row: ", err.Error())
		return err
	}

	err = insertLogEntry(db, newSessionId, filePath)
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

func main() {
	connectionString := fmt.Sprintf("Data Source=%s;Initial Catalog=%s;Integrated Security=True;Trust Server Certificate=True", dataSource, initialCatalog)

	db, err := sql.Open("mssql", connectionString)
	if err != nil {
		log.Fatal("Error creating connection pool: ", err.Error())
	}
	defer db.Close()

	fmt.Print("Checking for new programs...")
	newProgramsCount, err := insertNewPrograms(db, logsRootPath)
	if err != nil {
		fmt.Println("Error: ", err.Error())
	}
	fmt.Printf(" %d\n\n", newProgramsCount)

	err = insertNewLogs(db)
	if err != nil {
		log.Fatal("Error inserting new logs: ", err.Error())
	}

	fmt.Println("\nAll done")
}

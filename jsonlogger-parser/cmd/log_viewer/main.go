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
	id              int64
	name            string
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

// var ValidTableNames = map[string]struct{}{
// 	"log_entries":  {},
// 	"log_sessions": {},
// 	"programs":     {},
// }

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
			&prog.id, &prog.name, &prog.log_folder_path,
			&prog.archive_days, &prog.delete_days); err != nil {
			return nil, err
		}
		programs = append(programs, prog)
	}

	return programs, nil
}

func getSessions(db *sql.DB) ([]Session, error) {
	sessionsQuery := "select * from log_sessions"
	rows, err := db.Query(sessionsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var sesh Session
		if err := rows.Scan(
			&sesh.id, &sesh.program_id, &sesh.has_warning, &sesh.has_error,
			&sesh.has_fatal, &sesh.created_date, &sesh.is_archived); err != nil {
			return nil, err
		}
		sessions = append(sessions, sesh)
	}

	return sessions, nil
}

// func scanRowsToStruct[T any](rows *sql.Rows, item *T) error {

// 	columns, err := rows.Columns()
// 	if err != nil {
// 		return err
// 	}

// 	values := make([]any, len(columns))
// 	for i := range values {
// 		values[i] = new(sql.NullString)
// 	}
// 	if err := rows.Scan(values...); err != nil {
// 		return err
// 	}

// 	v := reflect.ValueOf(item).Elem()
// 	for i, col := range columns {
// 		field := v.FieldByName(col)
// 		if field.IsValid() && field.CanSet() {
// 			value := reflect.ValueOf(values[i]).Elem()

// 			if sqlValue, ok := value.Interface().(driver.Valuer); ok {
// 				val, err := sqlValue.Value()
// 				if err == nil && val != nil {
// 					field.Set(reflect.ValueOf(val))
// 				}
// 			}
// 		}
// 	}

// 	return nil

// 	// scanValues := make(T, value.NumField())
// 	// for i := range scanValues {
// 	// 	scanValues[i] = value.Field(i).Addr().Interface()
// 	// }

// 	// if err := rows.Scan(scanValues...); err != nil {
// 	// 	return err
// 	// }

// 	// return nil
// }

// func selectAllFromTable[T any](db *sql.DB, tableName string) ([]T, error) {
// 	if _, ok := ValidTableNames[tableName]; !ok {
// 		//return reflect.Zero(reflect.SliceOf(itemType)),
// 		return nil,
// 			errors.New(fmt.Sprintf("'%s' is not a valid table name", tableName))
// 	}
// 	programsQuery := fmt.Sprintf("select * from %s", tableName)

// 	rows, err := db.Query(programsQuery)
// 	if err != nil {
// 		//return reflect.Zero(reflect.SliceOf(itemType)), err
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	//var items = reflect.MakeSlice(reflect.SliceOf(itemType), 0, 0)
// 	items := reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf((*T)(nil)).Elem()), 0, 0)
// 	log.Println("made slice")
// 	for rows.Next() {
// 		var item reflect.Value
// 		if err := scanRowsToStruct(rows, items.Interface()); err != nil {
// 			log.Println("Unable to parse row")
// 			//return reflect.Zero(reflect.SliceOf(itemType)), err
// 			return nil, err
// 		}
// 		items = reflect.Append(items, item)
// 	}

// 	return items.Interface().([]T), nil
// }

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file: ", err.Error())
	}
}

func main() {

	// programs, err := getPrograms(db)
	// if err != nil {
	// 	log.Println("Error getting programs: ", err.Error())
	// } else {
	// 	fmt.Printf("%+v\n\n", programs)
	// }

	// sessions, err := getSessions(db)
	// if err != nil {
	// 	log.Println("Error getting sessions: ", err.Error())
	// } else {
	// 	fmt.Printf("%+v\n\n", sessions)
	// }

	http.HandleFunc("/", BaseHandler)
	http.HandleFunc("/programs", ProgramsHandler)
	log.Fatal(http.ListenAndServe(":8000", nil))
}

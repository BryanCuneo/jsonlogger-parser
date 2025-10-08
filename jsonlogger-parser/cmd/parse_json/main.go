package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

type PsTimestamp struct {
	time.Time
}

// UnmarshalJSON method for PsTimestamp
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

type LogEntry struct {
	Timestamp  PsTimestamp `json:"timestamp"`
	Level      string      `json:"level"`
	Message    string      `json:"message"`
	Context    string      `json:"context,omitempty"`
	CalledFrom string      `json:"calledFrom"`
	CallStack  string      `json:"callStack,omitempty"`
}

// Custom JSON unmarshalling is required so that we can throw an error when
// "CalledFrom" is empty
func (l *LogEntry) UnmarshalJSON(data []byte) error {
	temp := &struct {
		Timestamp  PsTimestamp `json:"timestamp"`
		Level      string      `json:"level"`
		Message    string      `json:"message"`
		Context    string      `json:"context,omitempty"`
		CalledFrom string      `json:"calledFrom"`
		CallStack  string      `json:"callStack,omitempty"`
	}{}

	if err := json.Unmarshal(data, temp); err != nil {
		return err
	}

	if temp.CalledFrom == "" {
		return errors.New("calledFrom cannot be empty")
	}

	l.Timestamp = temp.Timestamp
	l.Level = temp.Level
	l.Message = temp.Message
	l.Context = temp.Context
	l.CalledFrom = temp.CalledFrom
	l.CallStack = temp.CallStack

	return nil
}

type FinalEntry struct {
	Timestamp PsTimestamp `json:"timestamp"`
	Level     string      `json:"level"`
	Message   string      `json:"Message,omitempty"`
}

func main() {
	var initialEntry InitialEntry
	var logEntries []LogEntry
	var finalEntry FinalEntry

	if len(os.Args) < 2 {
		fmt.Println("Error: please provide a filepath as an argument")
		os.Exit(1)
	}

	file, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Parse initial [START] log entry
	scanner.Scan()
	firstEntry := scanner.Bytes()

	// Check and remove BOM if present
	if after, ok := bytes.CutPrefix(firstEntry, []byte{0xEF, 0xBB, 0xBF}); ok {
		firstEntry = after
	}

	if err := json.Unmarshal(firstEntry, &initialEntry); err != nil {
		fmt.Println("Error unmarshalling:", err)
		return
	}

	if initialEntry.HasWarning {
		fmt.Println("Has warnings")
	} else {
		fmt.Println("No warnings")
	}

	if initialEntry.HasError {
		fmt.Println("Has errors")
	} else {
		fmt.Println("No errors")
	}

	if initialEntry.HasFatal {
		fmt.Println("Has fatal error")
	} else {
		fmt.Println("No fatal error")
	}

	fmt.Printf("Initial Entry: %+v\n", initialEntry)

	for scanner.Scan() {
		var entry LogEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err == nil {
			logEntries = append(logEntries, entry)
		} else {
			// Try to parse an [END] log entry
			if err := json.Unmarshal(scanner.Bytes(), &finalEntry); err != nil {
				fmt.Println("Error unmarshalling end entry:", err)
				return
			}
		}
	}

	fmt.Printf("Found %d log entries\n", len(logEntries))
	if finalEntry.Level != "" {
		fmt.Printf("Final Entry: %+v\n\n", finalEntry)
	} else {
		fmt.Println("No final entry in this log file")
	}

	if err = scanner.Err(); err != nil {
		fmt.Println("Error reading from scanner:", err)
	}
}

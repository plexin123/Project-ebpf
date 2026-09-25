package database

import (
	"database/sql"
	"log"

	"ebpf-project/backend/collector"

	_ "modernc.org/sqlite"
)

// create structure to contain the db structure
// create function that open the database, and execute the creation of a table
// return the structure

type Database struct {
	db *sql.DB
}

func Open(path string) (*Database, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		log.Fatalf("Was not able to open database", err)
	}

	schema := "IDK :("

	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}

	return &Database{
		db: db,
	}, nil
}

//	type FunctionEvent struct {
//		FuncName string  `json:"funcName"`
//		Duration uint64  `json:"duration"`
//		Status   Status  `json:"status"`
//		Baseline uint64  `json:"baseline"`
//		Current  uint64  `json:"current"`
//		DriftPct float64 `json:"driftPct"`
//	}
func (s *Database) InsertEvent(functionEvent collector.FunctionEvent) error {
	// TO DO
	// INSERT INTO TABLE
	// IF NOT RETURN ERROR
	return nil
}

func (s *Database) GetHistory(functionName string) ([]collector.FunctionEvent, error) {
	// TO DO
	// with a functionName
	// return a history of this function
	return nil, nil
}

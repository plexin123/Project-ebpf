package database

import (
	"database/sql"
	"log"
	"time"

	"ebpf-project/backend/collector"

	_ "modernc.org/sqlite"

	"context"
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

// INSERTING A NEW EVENT WHEN IT ARRIVES

func (s *Database) InsertEvent(functionEvent collector.FunctionEvent) (int64, error) {
	query := "INSERT INTO EVENT (FunctionName,Baseline,...) VALUES(? ,?)"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := s.db.ExecContext(ctx, query, functionEvent.FuncName, functionEvent.Baseline, functionEvent.Status, functionEvent.DriftPct, functionEvent.Current)
	if err != nil {
		log.Fatalf("There has been an error executing the query", err)
	}
	last_id, err := result.LastInsertId()
	if err != nil {
		log.Fatalf("Could not retrieve the last id", err)
	}
	return last_id, nil
}

func (s *Database) GetHistory(functionName string, rows int) ([]collector.FunctionEvent, error) {
	query := "SELECT * FROM EVENT WHERE (FunctionName) EQUAL VALUE(?)"
	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()
	result, err := s.db.QueryContext(ctx, query, functionName)
	if err != nil {
		log.Fatalf("Could not retrieve the last id", err)
	}
	var functionEvents []collector.FunctionEvent
	for result.Next() {
		var functionEvent collector.FunctionEvent
		err := result.Scan(&functionEvent.FuncName, &functionEvent.Duration)
		if err != nil {
			log.Fatalf("Could not retrieve the last id", err)
		}
		functionEvents = append(functionEvents, functionEvent)
	}

	return functionEvents, nil
}

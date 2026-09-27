package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "modernc.org/sqlite"

	"context"
)

type Database struct {
	db *sql.DB
}

type FunctionEvent struct {
	FuncName  string  `json:"funcName"`
	Commit    string  `json:"commit"`
	Duration  uint64  `json:"duration"`
	Status    Status  `json:"status"`
	Baseline  uint64  `json:"baseline"`
	Current   uint64  `json:"current"`
	DriftPct  float64 `json:"driftPct"`
	TimeStamp uint64  `json:"timestamp"`
}

type Status string

const (
	StatusOk          Status = "ok"
	StatusBaselineSet Status = "baseline_set"
	StatusRegression  Status = "regression"
)

func Open(path string) (*Database, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		log.Fatalf("Was not able to open database %v", err)
	}

	schema := `CREATE TABLE IF NOT EXISTS function_event(
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				function_name TEXT NOT NULL,
				commit_hash TEXT NOT NULL,
				duration INTEGER NOT NULL,
				status TEXT NOT NULL,
				baseline INTEGER NOT NULL,
				current INTEGER  NOT NULL,
				drift_pct REAL NOT NULL,
				time_stamp INTEGER NOT NULL)`

	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}

	return &Database{
		db: db,
	}, nil
}

// INSERTING A NEW EVENT WHEN IT ARRIVES

func (s *Database) InsertEvent(functionEvent FunctionEvent) (int64, error) {
	query := "INSERT INTO function_event (function_name,commit_hash,duration,status,baseline,current,drift_pct,time_stamp) VALUES(?,?,?,?,?,?,?,?)"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := s.db.ExecContext(ctx, query, functionEvent.FuncName, functionEvent.Baseline, functionEvent.Status, functionEvent.DriftPct, functionEvent.Current)
	if err != nil {
		return -1, fmt.Errorf("Error inserting event: %w", err)
	}
	last_id, err := result.LastInsertId()
	if err != nil {
		return -1, fmt.Errorf("Error inserting event: %w", err)
	}
	return last_id, nil
}

// GetHistory of a function -> from one specific time stamp
// Or just a fix amount of rows mm
func (s *Database) GetHistory(functionName string, rows int) ([]FunctionEvent, error) {
	query := "SELECT * FROM function_event WHERE function_name = ? ORDER BY time_stamp DESC LIMIT ?"
	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()
	result, err := s.db.QueryContext(ctx, query, functionName, rows)
	if err != nil {
		return nil, fmt.Errorf("Error inserting event: %w", err)
	}
	var functionEvents []FunctionEvent
	for result.Next() {
		var functionEvent FunctionEvent
		err := result.Scan(&functionEvent.FuncName, &functionEvent.TimeStamp, &functionEvent.Duration, &functionEvent.Baseline, &functionEvent.Current, &functionEvent.DriftPct, &functionEvent.Status)
		if err != nil {
			return nil, fmt.Errorf("Error adding events: %w", err)
		}
		functionEvents = append(functionEvents, functionEvent)
	}

	return functionEvents, nil
}

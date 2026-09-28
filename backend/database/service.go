package database

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

type Service struct {
	database *Database
}

func New(DatabaseInstance *Database) *Service {
	return &Service{
		database: DatabaseInstance,
	}
}

func (s *Service) GetHistoryByName(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	amount_of_rows, err := strconv.Atoi(r.URL.Query().Get("rows"))
	if err != nil {
		fmt.Printf("Could not convert into integer")
		return
	}
	functionEvents, err := s.database.GetHistory(name, amount_of_rows)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if err != nil {
		error_message := map[string]string{
			"error": "Failed to retrieve history",
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(error_message)
		return
	}
	json.NewEncoder(w).Encode(functionEvents)
}

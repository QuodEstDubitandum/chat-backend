package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"realtime-chat/database"
)

func GetLastMessages(w http.ResponseWriter, r *http.Request){
	apiKey := r.Header.Get("X-API-Key")
	if apiKey != os.Getenv("API_KEY") {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`Invalid API Key`))
		return
	}
	ctx := context.Background()
	dbClient := database.GetDbClient()
	res := database.GetLatestChats(ctx, dbClient)

	// Convert the array of structs to JSON
	jsonData, err := json.Marshal(res)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to marshal JSON"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}
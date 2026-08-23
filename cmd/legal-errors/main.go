package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/example/legal-matter-error-capture/internal/infrai"
	"github.com/example/legal-matter-error-capture/internal/legalflow"
)

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}
	client := infrai.New(key)

	http.HandleFunc("POST /capture", func(w http.ResponseWriter, r *http.Request) {
		var failure legalflow.Failure
		if err := json.NewDecoder(r.Body).Decode(&failure); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		capture, err := legalflow.BuildCapture(failure)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		data, err := client.Capture(r.Context(), failure.EventID, capture)
		if err != nil {
			var apiErr *infrai.Error
			if errors.As(err, &apiErr) && apiErr.Status >= 400 && apiErr.Status < 500 {
				writeJSON(w, apiErr.Status, map[string]string{"error": apiErr.Error()})
				return
			}
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "error capture request failed"})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"captured": true, "data": data})
	})

	log.Println("legal error capture listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode response: %v", err)
	}
}

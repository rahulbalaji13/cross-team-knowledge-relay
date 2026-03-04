package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type createBountyRequest struct {
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Skills       []string `json:"skills"`
	BountyAmount int      `json:"bounty_amount"`
	TTLSeconds   int      `json:"ttl_seconds"`
}

type bountyResponse struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Skills      []string `json:"skills"`
	Amount      int      `json:"amount"`
	ExpiresAt   string   `json:"expiresAt"`
	Status      string   `json:"status"`
}

var (
	bounties  = make([]bountyResponse, 0)
	bountyMux sync.RWMutex
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", withCORS(rootHandler))
	mux.HandleFunc("/api/v1/health", withCORS(healthHandler))
	mux.HandleFunc("/api/v1/bounties", withCORS(bountiesHandler))
	mux.HandleFunc("/api/v1/escrow/", withCORS(releaseEscrowHandler))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting API server on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

func withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, X-Idempotency-Key")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func rootHandler(w http.ResponseWriter, _ *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "online", "service": "Cross-Team Knowledge Relay API"})
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func bountiesHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		bountyMux.RLock()
		defer bountyMux.RUnlock()
		json.NewEncoder(w).Encode(map[string]any{"bounties": bounties})
	case http.MethodPost:
		var req createBountyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(req.Title) == "" || len(req.Skills) == 0 || req.BountyAmount < 1 || req.TTLSeconds < 60 {
			http.Error(w, `{"error":"missing required fields"}`, http.StatusBadRequest)
			return
		}

		newBounty := bountyResponse{
			ID:          time.Now().UTC().Format("20060102150405.000000000"),
			Title:       req.Title,
			Description: req.Description,
			Skills:      req.Skills,
			Amount:      req.BountyAmount,
			ExpiresAt:   time.Now().UTC().Add(time.Duration(req.TTLSeconds) * time.Second).Format(time.RFC3339),
			Status:      "OPEN",
		}

		bountyMux.Lock()
		bounties = append([]bountyResponse{newBounty}, bounties...)
		bountyMux.Unlock()

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]any{"status": "accepted", "message": "Bounty matching started", "bounty": newBounty})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
	}
}

func releaseEscrowHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
		return
	}
	if !strings.HasSuffix(r.URL.Path, "/release") {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
		return
	}
	if r.Header.Get("X-Idempotency-Key") == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Missing Idempotency-Key"})
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "released"})
}

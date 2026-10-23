package main

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"os"
	"sort"
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
	PosterTeam   string   `json:"poster_team"`
}

type expert struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Team       string            `json:"team"`
	Reputation float64           `json:"reputation"`
	Skills     map[string]int    `json:"skills"`
	LastActive map[string]string `json:"last_active"`
}

type matchedExpert struct {
	ExpertID      string   `json:"expert_id"`
	Name          string   `json:"name"`
	Team          string   `json:"team"`
	Score         float64  `json:"score"`
	MatchedSkills []string `json:"matched_skills"`
}

type bountyResponse struct {
	ID             string          `json:"id"`
	Title          string          `json:"title"`
	Description    string          `json:"description"`
	Skills         []string        `json:"skills"`
	Amount         int             `json:"amount"`
	ExpiresAt      string          `json:"expiresAt"`
	Status         string          `json:"status"`
	PosterTeam     string          `json:"posterTeam"`
	MatchedExperts []matchedExpert `json:"matchedExperts"`
}

var (
	bounties  = make([]bountyResponse, 0)
	bountyMux sync.RWMutex

	experts = []expert{
		{ID: "eng-101", Name: "Maya Chen", Team: "Payments", Reputation: 4.8, Skills: map[string]int{"go": 5, "distributed systems": 4, "kafka": 5}, LastActive: map[string]string{"go": "2026-02-26", "distributed systems": "2026-02-20", "kafka": "2026-02-28"}},
		{ID: "eng-102", Name: "Arjun Patel", Team: "Infra", Reputation: 4.6, Skills: map[string]int{"neo4j": 5, "graph": 5, "go": 4}, LastActive: map[string]string{"neo4j": "2026-02-27", "graph": "2026-02-25", "go": "2026-02-24"}},
		{ID: "eng-103", Name: "Priya Nair", Team: "Search", Reputation: 4.7, Skills: map[string]int{"recommendation": 5, "python": 5, "distributed systems": 4}, LastActive: map[string]string{"recommendation": "2026-02-23", "python": "2026-02-28", "distributed systems": "2026-02-22"}},
		{ID: "eng-104", Name: "Luca Romano", Team: "Commerce", Reputation: 4.5, Skills: map[string]int{"frontend": 5, "react": 5, "next.js": 5, "typescript": 4}, LastActive: map[string]string{"frontend": "2026-02-27", "react": "2026-02-27", "next.js": "2026-02-26", "typescript": "2026-02-26"}},
		{ID: "eng-105", Name: "Sara Ali", Team: "Platform", Reputation: 4.9, Skills: map[string]int{"go": 5, "neo4j": 4, "redis": 5, "distributed systems": 5}, LastActive: map[string]string{"go": "2026-02-28", "neo4j": "2026-02-21", "redis": "2026-02-28", "distributed systems": "2026-02-27"}},
	}
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", withCORS(rootHandler))
	mux.HandleFunc("/api/v1/health", withCORS(healthHandler))
	mux.HandleFunc("/api/v1/experts", withCORS(expertsHandler))
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

func expertsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"experts": experts})
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

		posterTeam := strings.TrimSpace(req.PosterTeam)
		if posterTeam == "" {
			posterTeam = "Unknown"
		}

		cleanedSkills := normalizeSkills(req.Skills)
		matches := computeMatches(cleanedSkills, posterTeam)

		newBounty := bountyResponse{
			ID:             time.Now().UTC().Format("20060102150405.000000000"),
			Title:          req.Title,
			Description:    req.Description,
			Skills:         cleanedSkills,
			Amount:         req.BountyAmount,
			ExpiresAt:      time.Now().UTC().Add(time.Duration(req.TTLSeconds) * time.Second).Format(time.RFC3339),
			Status:         "OPEN",
			PosterTeam:     posterTeam,
			MatchedExperts: matches,
		}

		bountyMux.Lock()
		bounties = append([]bountyResponse{newBounty}, bounties...)
		bountyMux.Unlock()

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]any{"status": "accepted", "message": "Bounty matching completed", "bounty": newBounty})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
	}
}

func computeMatches(bountySkills []string, posterTeam string) []matchedExpert {
	candidates := make([]matchedExpert, 0)
	for _, ex := range experts {
		if strings.EqualFold(ex.Team, posterTeam) {
			continue
		}

		var score float64
		matched := make([]string, 0)
		for _, s := range bountySkills {
			normalized := strings.ToLower(strings.TrimSpace(s))
			level, ok := ex.Skills[normalized]
			if !ok {
				continue
			}
			matched = append(matched, s)

			skillScore := float64(level) / 5.0
			recencyScore := decay(ex.LastActive[normalized])
			score += (skillScore * 0.6) + (recencyScore * 0.4)
		}

		if len(matched) == 0 {
			continue
		}

		score = score + (ex.Reputation / 5.0)
		candidates = append(candidates, matchedExpert{
			ExpertID:      ex.ID,
			Name:          ex.Name,
			Team:          ex.Team,
			Score:         math.Round(score*100) / 100,
			MatchedSkills: matched,
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	if len(candidates) > 3 {
		return candidates[:3]
	}
	return candidates
}

func normalizeSkills(skills []string) []string {
	result := make([]string, 0, len(skills))
	seen := map[string]bool{}
	for _, s := range skills {
		clean := strings.TrimSpace(s)
		if clean == "" {
			continue
		}
		key := strings.ToLower(clean)
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, clean)
	}
	return result
}

func decay(lastUsed string) float64 {
	if lastUsed == "" {
		return 0.5
	}
	t, err := time.Parse("2006-01-02", lastUsed)
	if err != nil {
		return 0.5
	}
	days := time.Since(t).Hours() / 24
	if days < 0 {
		days = 0
	}
	return math.Exp(-0.03 * days)
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

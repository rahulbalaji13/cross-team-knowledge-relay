package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	_ "github.com/lib/pq"
)

type createBountyRequest struct {
	Title string `json:"title"`
	Description string `json:"description"`
	Skills []string `json:"skills"`
	BountyAmount int `json:"bounty_amount"`
	TTLSeconds int `json:"ttl_seconds"`
	PosterTeam string `json:"poster_team"`
	CreatedBy string `json:"created_by"`
}
type expert struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Team string `json:"team"`
	Reputation float64 `json:"reputation"`
	Skills map[string]int `json:"skills"`
	LastActive map[string]string `json:"last_active"`
}
type matchedExpert struct {
	ExpertID string `json:"expert_id"`
	Name string `json:"name"`
	Team string `json:"team"`
	Score float64 `json:"score"`
	MatchedSkills []string `json:"matched_skills"`
}
type bountyResponse struct {
	ID string `json:"id"`
	Title string `json:"title"`
	Description string `json:"description"`
	Skills []string `json:"skills"`
	Amount int `json:"amount"`
	ExpiresAt string `json:"expiresAt"`
	Status string `json:"status"`
	PosterTeam string `json:"posterTeam"`
	CreatedBy string `json:"createdBy"`
	MatchedExperts []matchedExpert `json:"matchedExperts"`
	CreatedAt time.Time `json:"-"`
}
type app struct {
	db *sql.DB
	apiToken string
	allowedOrigins map[string]bool
	limits *rateLimiter
}
type rateLimiter struct { mu sync.Mutex; clients map[string]bucket }
type bucket struct { count int; reset time.Time }

var experts = []expert{
	{ID:"eng-101",Name:"Maya Chen",Team:"Payments",Reputation:4.8,Skills:map[string]int{"go":5,"distributed systems":4,"kafka":5},LastActive:map[string]string{"go":"2026-02-26","distributed systems":"2026-02-20","kafka":"2026-02-28"}},
	{ID:"eng-102",Name:"Arjun Patel",Team:"Infra",Reputation:4.6,Skills:map[string]int{"neo4j":5,"graph":5,"go":4},LastActive:map[string]string{"neo4j":"2026-02-27","graph":"2026-02-25","go":"2026-02-24"}},
	{ID:"eng-103",Name:"Priya Nair",Team:"Search",Reputation:4.7,Skills:map[string]int{"recommendation":5,"python":5,"distributed systems":4},LastActive:map[string]string{"recommendation":"2026-02-23","python":"2026-02-28","distributed systems":"2026-02-22"}},
	{ID:"eng-104",Name:"Luca Romano",Team:"Commerce",Reputation:4.5,Skills:map[string]int{"frontend":5,"react":5,"next.js":5,"typescript":4},LastActive:map[string]string{"frontend":"2026-02-27","react":"2026-02-27","next.js":"2026-02-26","typescript":"2026-02-26"}},
	{ID:"eng-105",Name:"Sara Ali",Team:"Platform",Reputation:4.9,Skills:map[string]int{"go":5,"neo4j":4,"redis":5,"distributed systems":5},LastActive:map[string]string{"go":"2026-02-28","neo4j":"2026-02-21","redis":"2026-02-28","distributed systems":"2026-02-27"}},
}

func main() {
	dsn := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dsn == "" { log.Fatal("DATABASE_URL is required") }
	db, err := sql.Open("postgres", dsn)
	if err != nil { log.Fatal(err) }
	defer db.Close()
	db.SetMaxOpenConns(20); db.SetMaxIdleConns(5); db.SetConnMaxLifetime(30*time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil { log.Fatalf("database connection: %v", err) }
	if _, err := db.ExecContext(ctx, schema); err != nil { log.Fatalf("database schema: %v", err) }

	origins := map[string]bool{}
	for _, origin := range strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",") {
		if value := strings.TrimSpace(origin); value != "" { origins[value] = true }
	}
	a := &app{db:db, apiToken:os.Getenv("API_BEARER_TOKEN"), allowedOrigins:origins, limits:&rateLimiter{clients:map[string]bucket{}}}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", a.health)
	mux.HandleFunc("/api/v1/bounties", a.bounties)
	mux.HandleFunc("/api/v1/events", a.events)
	mux.HandleFunc("/", a.root)
	server := &http.Server{Addr:":"+env("PORT","8080"), Handler:a.middleware(mux), ReadHeaderTimeout:5*time.Second, ReadTimeout:15*time.Second, IdleTimeout:60*time.Second}
	stop, stopCancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopCancel()
	go func(){ <-stop.Done(); ctx, cancel := context.WithTimeout(context.Background(),10*time.Second); defer cancel(); _ = server.Shutdown(ctx) }()
	log.Printf("API listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err,http.ErrServerClosed) { log.Fatal(err) }
}

const schema = `CREATE TABLE IF NOT EXISTS bounties (
 id TEXT PRIMARY KEY,
 title TEXT NOT NULL,
 description TEXT NOT NULL DEFAULT '',
 skills JSONB NOT NULL,
 amount INTEGER NOT NULL CHECK (amount > 0),
 expires_at TIMESTAMPTZ NOT NULL,
 status TEXT NOT NULL DEFAULT 'OPEN',
 poster_team TEXT NOT NULL,
 created_by TEXT NOT NULL DEFAULT 'demo',
 matched_experts JSONB NOT NULL DEFAULT '[]'::jsonb,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
ALTER TABLE bounties ADD COLUMN IF NOT EXISTS created_by TEXT NOT NULL DEFAULT 'demo';
CREATE INDEX IF NOT EXISTS bounties_created_at_idx ON bounties (created_at DESC);
CREATE INDEX IF NOT EXISTS bounties_expires_at_idx ON bounties (expires_at);`

func env(key, fallback string) string { if v:=strings.TrimSpace(os.Getenv(key)); v!="" { return v }; return fallback }
func (a *app) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request) {
		w.Header().Set("X-Content-Type-Options","nosniff")
		w.Header().Set("Referrer-Policy","no-referrer")
		w.Header().Set("Cache-Control","no-store")
		origin:=r.Header.Get("Origin")
		if origin!="" && a.allowedOrigins[origin] { w.Header().Set("Access-Control-Allow-Origin",origin); w.Header().Add("Vary","Origin") }
		w.Header().Set("Access-Control-Allow-Headers","Authorization, Content-Type, Last-Event-ID")
		w.Header().Set("Access-Control-Allow-Methods","GET, POST, OPTIONS")
		if r.Method==http.MethodOptions { w.WriteHeader(http.StatusNoContent); return }
		if !a.limits.allow(clientIP(r)) { writeJSON(w,http.StatusTooManyRequests,map[string]string{"error":"rate limit exceeded"}); return }
		next.ServeHTTP(w,r)
	})
}
func clientIP(r *http.Request) string { host,_,err:=net.SplitHostPort(r.RemoteAddr); if err==nil { return host }; return r.RemoteAddr }
func (l *rateLimiter) allow(ip string) bool {
	l.mu.Lock(); defer l.mu.Unlock()
	now:=time.Now(); b,ok:=l.clients[ip]
	if !ok || now.After(b.reset) { l.clients[ip]=bucket{count:1,reset:now.Add(time.Minute)}; return true }
	if b.count>=120 { return false }; b.count++; l.clients[ip]=b; return true
}
func (a *app) root(w http.ResponseWriter,r *http.Request) { if r.URL.Path!="/" { http.NotFound(w,r); return }; writeJSON(w,http.StatusOK,map[string]string{"service":"Cross-Team Knowledge Relay API","status":"online"}) }
func (a *app) health(w http.ResponseWriter,r *http.Request) {
	ctx,cancel:=context.WithTimeout(r.Context(),2*time.Second); defer cancel()
	if err:=a.db.PingContext(ctx); err!=nil { writeJSON(w,http.StatusServiceUnavailable,map[string]string{"status":"unavailable"}); return }
	writeJSON(w,http.StatusOK,map[string]string{"status":"ok","database":"ok"})
}
func (a *app) bounties(w http.ResponseWriter,r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rows,err:=a.db.QueryContext(r.Context(),`SELECT id,title,description,skills,amount,expires_at,status,poster_team,created_by,matched_experts,created_at FROM bounties WHERE expires_at > now() ORDER BY created_at DESC LIMIT 100`)
		if err!=nil { writeJSON(w,http.StatusInternalServerError,map[string]string{"error":"could not load bounties"}); return }; defer rows.Close()
		items:=make([]bountyResponse,0)
		for rows.Next() { item,err:=scanBounty(rows); if err!=nil { writeJSON(w,http.StatusInternalServerError,map[string]string{"error":"could not read bounties"}); return }; items=append(items,item) }
		if err:=rows.Err(); err!=nil { writeJSON(w,http.StatusInternalServerError,map[string]string{"error":"could not read bounties"}); return }
		writeJSON(w,http.StatusOK,map[string]any{"bounties":items})
	case http.MethodPost:
		if a.apiToken=="" { writeJSON(w,http.StatusServiceUnavailable,map[string]string{"error":"API_BEARER_TOKEN is not configured"}); return }
		if r.Header.Get("Authorization")!="Bearer "+a.apiToken { writeJSON(w,http.StatusUnauthorized,map[string]string{"error":"unauthorized"}); return }
		r.Body=http.MaxBytesReader(w,r.Body,64<<10)
		var req createBountyRequest
		dec:=json.NewDecoder(r.Body); dec.DisallowUnknownFields()
		if err:=dec.Decode(&req); err!=nil { writeJSON(w,http.StatusBadRequest,map[string]string{"error":"invalid JSON request"}); return }
		req.Title=strings.TrimSpace(req.Title); req.Description=strings.TrimSpace(req.Description); req.PosterTeam=strings.TrimSpace(req.PosterTeam)
		skills:=normalizeSkills(req.Skills)
		if len(req.Title)<3 || len(req.Title)>200 || len(req.Description)>4000 || len(skills)==0 || len(skills)>20 || req.BountyAmount<1 || req.BountyAmount>100000 || req.TTLSeconds<60 || req.TTLSeconds>30*24*3600 {
			writeJSON(w,http.StatusBadRequest,map[string]string{"error":"title (3-200 chars), description (max 4000), 1-20 skills, amount (1-100000), and TTL (60-2592000 seconds) are required"}); return
		}
		if req.PosterTeam=="" { req.PosterTeam="Unknown" }
		if req.CreatedBy=="" { writeJSON(w,http.StatusBadRequest,map[string]string{"error":"authenticated user is required"}); return }
		for _,skill:=range skills { if len(skill)>80 { writeJSON(w,http.StatusBadRequest,map[string]string{"error":"skill names must be 80 characters or fewer"}); return } }
		matches:=computeMatches(skills,req.PosterTeam)
		id,err:=newID(); if err!=nil { writeJSON(w,http.StatusInternalServerError,map[string]string{"error":"could not create bounty"}); return }
		now:=time.Now().UTC(); expires:=now.Add(time.Duration(req.TTLSeconds)*time.Second)
		skillsJSON,_:=json.Marshal(skills); matchesJSON,_:=json.Marshal(matches)
		_,err=a.db.ExecContext(r.Context(),`INSERT INTO bounties(id,title,description,skills,amount,expires_at,status,poster_team,created_by,matched_experts,created_at) VALUES($1,$2,$3,$4,$5,$6,'OPEN',$7,$8,$9,$10)`,id,req.Title,req.Description,skillsJSON,req.BountyAmount,expires,req.PosterTeam,req.CreatedBy,matchesJSON,now)
		if err!=nil { log.Printf("insert bounty: %v",err); writeJSON(w,http.StatusInternalServerError,map[string]string{"error":"could not save bounty"}); return }
		writeJSON(w,http.StatusCreated,map[string]any{"status":"created","bounty":bountyResponse{ID:id,Title:req.Title,Description:req.Description,Skills:skills,Amount:req.BountyAmount,ExpiresAt:expires.Format(time.RFC3339),Status:"OPEN",PosterTeam:req.PosterTeam,CreatedBy:req.CreatedBy,MatchedExperts:matches,CreatedAt:now}})
	default: writeJSON(w,http.StatusMethodNotAllowed,map[string]string{"error":"method not allowed"})
	}
}
type scanner interface { Scan(...any) error }
func scanBounty(row scanner) (bountyResponse,error) {
	var b bountyResponse; var skills,matches []byte
	err:=row.Scan(&b.ID,&b.Title,&b.Description,&skills,&b.Amount,&b.ExpiresAt,&b.Status,&b.PosterTeam,&b.CreatedBy,&matches,&b.CreatedAt)
	if err!=nil { return b,err }; if err=json.Unmarshal(skills,&b.Skills); err!=nil { return b,err }; if err=json.Unmarshal(matches,&b.MatchedExperts); err!=nil { return b,err }
	return b,nil
}
func (a *app) events(w http.ResponseWriter,r *http.Request) {
	flusher,ok:=w.(http.Flusher); if !ok { writeJSON(w,http.StatusInternalServerError,map[string]string{"error":"streaming is unavailable"}); return }
	cursor:=time.Now().UTC()
	if value:=r.Header.Get("Last-Event-ID"); value!="" { if t,err:=time.Parse(time.RFC3339Nano,value); err==nil { cursor=t } }
	w.Header().Set("Content-Type","text/event-stream"); w.Header().Set("Connection","keep-alive"); w.Header().Set("X-Accel-Buffering","no")
	fmt.Fprint(w,": connected\n\n"); flusher.Flush()
	ticker:=time.NewTicker(2*time.Second); defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done(): return
		case <-ticker.C:
			rows,err:=a.db.QueryContext(r.Context(),`SELECT id,title,description,skills,amount,expires_at,status,poster_team,matched_experts,created_at FROM bounties WHERE created_at > $1 ORDER BY created_at ASC LIMIT 100`,cursor)
			if err!=nil { fmt.Fprint(w,"event: error\ndata: {\"error\":\"stream temporarily unavailable\"}\n\n"); flusher.Flush(); continue }
			for rows.Next() {
				b,err:=scanBounty(rows); if err!=nil { continue }
				payload,err:=json.Marshal(b); if err!=nil { continue }
				fmt.Fprintf(w,"id: %s\nevent: bounty.created\ndata: %s\n\n",b.CreatedAt.Format(time.RFC3339Nano),payload)
				cursor=b.CreatedAt; flusher.Flush()
			}
			rows.Close()
		}
	}
}
func writeJSON(w http.ResponseWriter,status int,value any) { w.Header().Set("Content-Type","application/json"); w.WriteHeader(status); _=json.NewEncoder(w).Encode(value) }
func newID() (string,error) { var b [16]byte; if _,err:=rand.Read(b[:]); err!=nil { return "",err }; return hex.EncodeToString(b[:]),nil }
func normalizeSkills(skills []string) []string {
	result:=make([]string,0,len(skills)); seen:=map[string]bool{}
	for _,s:=range skills { clean:=strings.TrimSpace(s); key:=strings.ToLower(clean); if clean=="" || seen[key] { continue }; seen[key]=true; result=append(result,clean) }
	return result
}
func computeMatches(bountySkills []string,posterTeam string) []matchedExpert {
	candidates:=make([]matchedExpert,0)
	for _,ex:=range experts {
		if strings.EqualFold(ex.Team,posterTeam) { continue }
		var score float64; matched:=make([]string,0)
		for _,s:=range bountySkills {
			key:=strings.ToLower(strings.TrimSpace(s)); level,ok:=ex.Skills[key]; if !ok { continue }
			matched=append(matched,s); score+=(float64(level)/5*.6)+(decay(ex.LastActive[key])*.4)
		}
		if len(matched)==0 { continue }; score+=ex.Reputation/5
		candidates=append(candidates,matchedExpert{ExpertID:ex.ID,Name:ex.Name,Team:ex.Team,Score:math.Round(score*100)/100,MatchedSkills:matched})
	}
	sort.Slice(candidates,func(i,j int)bool{return candidates[i].Score>candidates[j].Score})
	if len(candidates)>3 { return candidates[:3] }; return candidates
}
func decay(lastUsed string) float64 {
	t,err:=time.Parse("2006-01-02",lastUsed); if err!=nil { return .5 }
	days:=time.Since(t).Hours()/24; if days<0 { days=0 }; return math.Exp(-.03*days)
}

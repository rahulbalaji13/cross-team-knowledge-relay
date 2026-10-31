package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	graph "github.com/rahulbalaji13/cross-team-knowledge-relay/backend/internal/infrastructure/neo4j"
	"github.com/redis/go-redis/v9"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const idempotencyTTL = 24 * time.Hour
const idempotencyLockTTL = 30 * time.Second
const bountyFeedCacheTTL = 30 * time.Second
const bountyFeedCacheOperationTimeout = 100 * time.Millisecond

const bountyFeedRevisionKey = "bounties:v1:feed:revision"
const bountyFeedCachePrefix = "bounties:v1:feed:"

var releaseIdempotencyLock = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0
`)

var readBountyFeedCache = redis.NewScript(`
local revision = redis.call("GET", KEYS[1])
if not revision then
  revision = "0"
  redis.call("SET", KEYS[1], revision)
end
local payload = redis.call("GET", ARGV[1] .. revision)
return {revision, payload or ""}
`)

type createBountyRequest struct {
	Title        string   `json:"title" binding:"required,min=3,max=200"`
	Description  string   `json:"description" binding:"max=4000"`
	Skills       []string `json:"skills" binding:"required,min=1,max=20"`
	BountyAmount int      `json:"bounty_amount" binding:"required,min=1,max=1000000"`
	TTLSeconds   int      `json:"ttl_seconds" binding:"required,min=60,max=2592000"`
	PosterTeam   string   `json:"poster_team" binding:"required,min=1,max=80"`
}

type bountyResponse struct {
	ID             string              `json:"id"`
	Title          string              `json:"title"`
	Description    string              `json:"description"`
	Skills         []string            `json:"skills"`
	Amount         int                 `json:"amount"`
	ExpiresAt      time.Time           `json:"expiresAt"`
	Status         string              `json:"status"`
	PosterTeam     string              `json:"posterTeam"`
	MatchedExperts []graph.ExpertMatch `json:"matchedExperts"`
	CreatedAt      time.Time           `json:"createdAt"`
}

type createBountyResponse struct {
	Status  string         `json:"status"`
	Message string         `json:"message"`
	Bounty  bountyResponse `json:"bounty"`
}

type bountyFeedResponse struct {
	Bounties []bountyResponse `json:"bounties"`
}

type bountyRecord struct {
	ID             string         `gorm:"primaryKey;size:36"`
	Title          string         `gorm:"size:200;not null"`
	Description    string         `gorm:"type:text"`
	Skills         datatypes.JSON `gorm:"type:jsonb;not null"`
	Amount         int            `gorm:"not null"`
	ExpiresAt      time.Time      `gorm:"index;not null"`
	Status         string         `gorm:"size:20;index;not null"`
	PosterTeam     string         `gorm:"size:80;not null"`
	MatchedExperts datatypes.JSON `gorm:"type:jsonb;not null"`
	CreatedAt      time.Time      `gorm:"index;not null"`
}

type idempotencyRecord struct {
	KeyHash      string         `gorm:"primaryKey;size:64"`
	RequestHash  string         `gorm:"size:64;not null"`
	ResponseJSON datatypes.JSON `gorm:"type:jsonb;not null"`
	CreatedAt    time.Time      `gorm:"not null"`
}

type expertMatcher interface {
	MatchExperts(context.Context, []string, string, int) ([]graph.ExpertMatch, error)
}

type api struct {
	db      *gorm.DB
	redis   *redis.Client
	matcher expertMatcher
}

type cachedResponse struct {
	RequestHash string          `json:"request_hash"`
	Response    json.RawMessage `json:"response"`
}

var errIdempotencyConflict = errors.New("idempotency key was already used with a different request")

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	databaseURL, err := requiredEnv("POSTGRES_URL")
	if err != nil {
		log.Fatal(err)
	}
	redisURL, err := requiredEnv("REDIS_URL")
	if err != nil {
		log.Fatal(err)
	}
	neo4jURI, err := requiredEnv("NEO4J_URI")
	if err != nil {
		log.Fatal(err)
	}
	neo4jUsername, err := requiredEnv("NEO4J_USERNAME")
	if err != nil {
		log.Fatal(err)
	}
	neo4jPassword, err := requiredEnv("NEO4J_PASSWORD")
	if err != nil {
		log.Fatal(err)
	}

	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{TranslateError: true})
	if err != nil {
		log.Fatalf("connect to PostgreSQL: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("get PostgreSQL connection pool: %v", err)
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		log.Fatalf("ping PostgreSQL: %v", err)
	}
	if err := db.AutoMigrate(&bountyRecord{}, &idempotencyRecord{}); err != nil {
		log.Fatalf("migrate PostgreSQL schema: %v", err)
	}

	redisOptions, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("parse REDIS_URL: %v", err)
	}
	redisClient := redis.NewClient(redisOptions)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("connect to Redis: %v", err)
	}

	graphRepository, err := graph.NewRepository(ctx, neo4jURI, neo4jUsername, neo4jPassword)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := graphRepository.Close(context.Background()); err != nil {
			log.Printf("close Neo4j connection: %v", err)
		}
	}()
	if err := graphRepository.EnsureSchema(ctx); err != nil {
		log.Fatal(err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	app := &api{db: db, redis: redisClient, matcher: graphRepository}
	router := app.routes()
	log.Printf("Starting Knowledge Relay API on :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("serve API: %v", err)
	}
}

func (a *api) routes() *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery(), corsMiddleware())
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "online", "service": "Cross-Team Knowledge Relay API"})
	})
	router.GET("/api/v1/health", a.healthHandler)
	router.GET("/api/v1/bounties", a.listBountiesHandler)
	router.POST("/api/v1/bounties", a.createBountyHandler)
	return router
}

func (a *api) healthHandler(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	sqlDB, err := a.db.DB()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "dependency": "postgres"})
		return
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		log.Printf("PostgreSQL health check failed: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "dependency": "postgres"})
		return
	}
	if err := a.redis.Ping(ctx).Err(); err != nil {
		log.Printf("Redis health check failed: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "dependency": "redis"})
		return
	}
	if checker, ok := a.matcher.(interface{ Ping(context.Context) error }); ok {
		if err := checker.Ping(ctx); err != nil {
			log.Printf("Neo4j health check failed: %v", err)
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "dependency": "neo4j"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (a *api) listBountiesHandler(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	cached, revision, cacheHit := a.cachedBountyFeed(ctx)
	if cacheHit {
		var response bountyFeedResponse
		if err := json.Unmarshal(cached, &response); err != nil {
			log.Printf("decode cached bounty feed: %v", err)
		} else if response.Bounties == nil {
			log.Printf("decode cached bounty feed: bounty list is null")
		} else {
			c.JSON(http.StatusOK, response)
			return
		}
	}

	now := time.Now().UTC()
	if err := a.db.WithContext(ctx).Model(&bountyRecord{}).
		Where("status = ? AND expires_at <= ?", "OPEN", now).
		Update("status", "EXPIRED").Error; err != nil {
		log.Printf("expire old bounties: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not refresh request status"})
		return
	}

	var records []bountyRecord
	if err := a.db.WithContext(ctx).
		Order("created_at DESC").Limit(100).Find(&records).Error; err != nil {
		log.Printf("load bounties: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not load requests"})
		return
	}
	responses := make([]bountyResponse, 0, len(records))
	for _, record := range records {
		response, err := record.response()
		if err != nil {
			log.Printf("decode bounty %s: %v", record.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "stored request data is invalid"})
			return
		}
		responses = append(responses, response)
	}
	feed := bountyFeedResponse{Bounties: responses}
	feedJSON, err := json.Marshal(feed)
	if err != nil {
		log.Printf("encode bounty feed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not encode requests"})
		return
	}
	if ttl := bountyFeedTTL(records, time.Now().UTC()); ttl > 0 && revision != "" {
		cacheCtx, cacheCancel := context.WithTimeout(ctx, bountyFeedCacheOperationTimeout)
		defer cacheCancel()
		if err := a.redis.Set(cacheCtx, bountyFeedCachePrefix+revision, feedJSON, ttl).Err(); err != nil {
			log.Printf("cache bounty feed: %v", err)
		}
	}
	c.Data(http.StatusOK, "application/json", feedJSON)
}

func (a *api) cachedBountyFeed(ctx context.Context) ([]byte, string, bool) {
	cacheCtx, cancel := context.WithTimeout(ctx, bountyFeedCacheOperationTimeout)
	defer cancel()
	values, err := readBountyFeedCache.Run(cacheCtx, a.redis, []string{bountyFeedRevisionKey}, bountyFeedCachePrefix).Slice()
	if err != nil {
		log.Printf("read bounty feed cache: %v", err)
		return nil, "", false
	}
	if len(values) != 2 {
		log.Printf("read bounty feed cache: unexpected Redis result length %d", len(values))
		return nil, "", false
	}
	revision, ok := redisString(values[0])
	if !ok {
		log.Printf("read bounty feed cache: invalid revision value")
		return nil, "", false
	}
	payload, ok := redisString(values[1])
	if !ok {
		log.Printf("read bounty feed cache: invalid payload value")
		return nil, revision, false
	}
	if payload == "" {
		return nil, revision, false
	}
	return []byte(payload), revision, true
}

func redisString(value any) (string, bool) {
	switch typed := value.(type) {
	case string:
		return typed, true
	case []byte:
		return string(typed), true
	default:
		return "", false
	}
}

func bountyFeedTTL(records []bountyRecord, now time.Time) time.Duration {
	ttl := bountyFeedCacheTTL
	for _, record := range records {
		if strings.EqualFold(record.Status, "OPEN") {
			untilExpiry := record.ExpiresAt.Sub(now)
			if untilExpiry < ttl {
				ttl = untilExpiry
			}
		}
	}
	if ttl <= 0 {
		return 0
	}
	return ttl
}

func (a *api) createBountyHandler(c *gin.Context) {
	var request createBountyRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload"})
		return
	}
	request.Title = strings.TrimSpace(request.Title)
	request.Description = strings.TrimSpace(request.Description)
	request.PosterTeam = strings.TrimSpace(request.PosterTeam)
	request.Skills = normalizeSkills(request.Skills)
	if utf8.RuneCountInString(request.Title) < 3 || request.PosterTeam == "" || len(request.Skills) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title, team, and at least one skill are required"})
		return
	}

	idempotencyKey := strings.TrimSpace(c.GetHeader("X-Idempotency-Key"))
	if idempotencyKey == "" || len(idempotencyKey) > 256 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Idempotency-Key is required and must be at most 256 characters"})
		return
	}
	requestJSON, err := json.Marshal(request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not encode request"})
		return
	}
	requestHash := hashString(string(requestJSON))
	keyHash := hashString(idempotencyKey)
	cacheKey := "idempotency:v1:" + keyHash
	lockValue := "processing:" + requestHash + ":" + uuid.NewString()
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	if cached, err := a.redis.Get(ctx, cacheKey).Result(); err == nil {
		if strings.HasPrefix(cached, "processing:") {
			processingHash, ok := processingRequestHash(cached)
			if !ok {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "idempotency service contains an invalid lock"})
				return
			}
			if processingHash != requestHash {
				c.JSON(http.StatusConflict, gin.H{"error": errIdempotencyConflict.Error()})
				return
			}
			if a.replyFromDatabase(ctx, c, keyHash, requestHash) {
				return
			}
			c.JSON(http.StatusConflict, gin.H{"error": "a request with this idempotency key is still in progress"})
			return
		}
		a.replyFromCache(c, cached, requestHash)
		return
	} else if !errors.Is(err, redis.Nil) {
		log.Printf("read Redis idempotency cache: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "idempotency service is unavailable"})
		return
	}

	acquired, err := a.redis.SetNX(ctx, cacheKey, lockValue, idempotencyLockTTL).Result()
	if err != nil {
		log.Printf("acquire Redis idempotency key: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "idempotency service is unavailable"})
		return
	}
	if !acquired {
		cached, err := a.redis.Get(ctx, cacheKey).Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			log.Printf("read in-progress Redis idempotency key: %v", err)
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "idempotency service is unavailable"})
			return
		}
		if err == nil && strings.HasPrefix(cached, "processing:") {
			processingHash, ok := processingRequestHash(cached)
			if !ok {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "idempotency service contains an invalid lock"})
				return
			}
			if processingHash != requestHash {
				c.JSON(http.StatusConflict, gin.H{"error": errIdempotencyConflict.Error()})
				return
			}
			if a.replyFromDatabase(ctx, c, keyHash, requestHash) {
				return
			}
			c.JSON(http.StatusConflict, gin.H{"error": "a request with this idempotency key is still in progress"})
			return
		}
		if err == nil {
			a.replyFromCache(c, cached, requestHash)
			return
		}
		c.JSON(http.StatusConflict, gin.H{"error": "a request with this idempotency key is still in progress"})
		return
	}

	keepCache := false
	defer func() {
		if !keepCache {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), time.Second)
			defer cleanupCancel()
			if err := releaseIdempotencyLock.Run(cleanupCtx, a.redis, []string{cacheKey}, lockValue).Err(); err != nil {
				log.Printf("release Redis idempotency key: %v", err)
			}
		}
	}()
	if a.replyFromDatabase(ctx, c, keyHash, requestHash) {
		keepCache = c.Writer.Status() == http.StatusOK
		return
	}

	matches, err := a.matcher.MatchExperts(ctx, request.Skills, request.PosterTeam, 3)
	if err != nil {
		log.Printf("match experts: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "expert matching is temporarily unavailable"})
		return
	}

	now := time.Now().UTC()
	response := bountyResponse{
		ID:             uuid.NewString(),
		Title:          request.Title,
		Description:    request.Description,
		Skills:         request.Skills,
		Amount:         request.BountyAmount,
		ExpiresAt:      now.Add(time.Duration(request.TTLSeconds) * time.Second),
		Status:         "OPEN",
		PosterTeam:     request.PosterTeam,
		MatchedExperts: matches,
		CreatedAt:      now,
	}
	result := createBountyResponse{
		Status: "created", Message: "Bounty saved and expert matches calculated",
		Bounty: response,
	}
	responseJSON, err := json.Marshal(result)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not encode request result"})
		return
	}
	record, err := newBountyRecord(response)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not encode request result"})
		return
	}

	err = a.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&record).Error; err != nil {
			return err
		}
		idempotency := idempotencyRecord{
			KeyHash: keyHash, RequestHash: requestHash,
			ResponseJSON: datatypes.JSON(responseJSON), CreatedAt: now,
		}
		return tx.Create(&idempotency).Error
	})
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) && a.replyFromDatabase(ctx, c, keyHash, requestHash) {
			keepCache = c.Writer.Status() == http.StatusOK
			return
		}
		log.Printf("persist bounty: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not save request"})
		return
	}

	cacheCtx, cacheCancel := context.WithTimeout(ctx, bountyFeedCacheOperationTimeout)
	defer cacheCancel()
	if err := a.redis.Incr(cacheCtx, bountyFeedRevisionKey).Err(); err != nil {
		log.Printf("invalidate bounty feed cache: %v", err)
	}

	cacheJSON, err := json.Marshal(cachedResponse{RequestHash: requestHash, Response: responseJSON})
	if err != nil {
		log.Printf("encode Redis idempotency result: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "request was saved but its idempotency result could not be encoded"})
		return
	}
	if err := a.redis.Set(ctx, cacheKey, cacheJSON, idempotencyTTL).Err(); err != nil {
		log.Printf("cache Redis idempotency result: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "request was saved; retry with the same idempotency key"})
		return
	}
	keepCache = true
	c.Data(http.StatusCreated, "application/json", responseJSON)
}

func (a *api) replyFromCache(c *gin.Context, cached, requestHash string) {
	var result cachedResponse
	if err := json.Unmarshal([]byte(cached), &result); err != nil {
		log.Printf("decode Redis idempotency result: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "idempotency cache contains invalid data"})
		return
	}
	if result.RequestHash != requestHash {
		c.JSON(http.StatusConflict, gin.H{"error": errIdempotencyConflict.Error()})
		return
	}
	c.Data(http.StatusOK, "application/json", result.Response)
}

func (a *api) replyFromDatabase(ctx context.Context, c *gin.Context, keyHash, requestHash string) bool {
	var result idempotencyRecord
	err := a.db.WithContext(ctx).First(&result, "key_hash = ?", keyHash).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false
	}
	if err != nil {
		log.Printf("read PostgreSQL idempotency record: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not verify request idempotency"})
		return true
	}
	if result.RequestHash != requestHash {
		c.JSON(http.StatusConflict, gin.H{"error": errIdempotencyConflict.Error()})
		return true
	}
	cacheJSON, err := json.Marshal(cachedResponse{
		RequestHash: result.RequestHash, Response: json.RawMessage(result.ResponseJSON),
	})
	if err != nil {
		log.Printf("encode PostgreSQL idempotency result: %v", err)
	} else if err := a.redis.Set(ctx, "idempotency:v1:"+keyHash, cacheJSON, idempotencyTTL).Err(); err != nil {
		log.Printf("refresh Redis idempotency cache: %v", err)
	}
	c.Data(http.StatusOK, "application/json", result.ResponseJSON)
	return true
}

func (r bountyRecord) response() (bountyResponse, error) {
	var skills []string
	if err := json.Unmarshal(r.Skills, &skills); err != nil {
		return bountyResponse{}, err
	}
	var matches []graph.ExpertMatch
	if err := json.Unmarshal(r.MatchedExperts, &matches); err != nil {
		return bountyResponse{}, err
	}
	return bountyResponse{
		ID: r.ID, Title: r.Title, Description: r.Description, Skills: skills,
		Amount: r.Amount, ExpiresAt: r.ExpiresAt, Status: r.Status,
		PosterTeam: r.PosterTeam, MatchedExperts: matches, CreatedAt: r.CreatedAt,
	}, nil
}

func newBountyRecord(response bountyResponse) (bountyRecord, error) {
	skillsJSON, err := json.Marshal(response.Skills)
	if err != nil {
		return bountyRecord{}, err
	}
	matchesJSON, err := json.Marshal(response.MatchedExperts)
	if err != nil {
		return bountyRecord{}, err
	}
	return bountyRecord{
		ID: response.ID, Title: response.Title, Description: response.Description,
		Skills: datatypes.JSON(skillsJSON), Amount: response.Amount, ExpiresAt: response.ExpiresAt,
		Status: response.Status, PosterTeam: response.PosterTeam,
		MatchedExperts: datatypes.JSON(matchesJSON), CreatedAt: response.CreatedAt,
	}, nil
}

func normalizeSkills(skills []string) []string {
	result := make([]string, 0, len(skills))
	seen := make(map[string]struct{}, len(skills))
	for _, skill := range skills {
		clean := strings.TrimSpace(skill)
		key := strings.ToLower(clean)
		if clean == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, clean)
	}
	sort.SliceStable(result, func(i, j int) bool {
		return strings.ToLower(result[i]) < strings.ToLower(result[j])
	})
	return result
}

func hashString(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func processingRequestHash(value string) (string, bool) {
	parts := strings.SplitN(value, ":", 3)
	if len(parts) != 3 || parts[0] != "processing" || parts[1] == "" || parts[2] == "" {
		return "", false
	}
	return parts[1], true
}

func corsMiddleware() gin.HandlerFunc {
	allowedOrigin := os.Getenv("FRONTEND_ORIGIN")
	if allowedOrigin == "" {
		allowedOrigin = "http://localhost:3000"
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == allowedOrigin {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, X-Idempotency-Key")
			c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func requiredEnv(name string) (string, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return "", fmt.Errorf("%s must be set", name)
	}
	return value, nil
}

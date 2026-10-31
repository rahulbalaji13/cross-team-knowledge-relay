package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	graph "github.com/rahulbalaji13/cross-team-knowledge-relay/backend/internal/infrastructure/neo4j"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type fixedMatcher struct {
	matches []graph.ExpertMatch
}

func (m fixedMatcher) MatchExperts(context.Context, []string, string, int) ([]graph.ExpertMatch, error) {
	return m.matches, nil
}

func TestNormalizeSkillsRemovesBlanksAndDuplicates(t *testing.T) {
	got := normalizeSkills([]string{" Go ", "go", "", "Neo4j", " neo4j "})
	want := []string{"Go", "Neo4j"}
	if len(got) != len(want) {
		t.Fatalf("normalizeSkills() = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("normalizeSkills() = %#v, want %#v", got, want)
		}
	}
}

func TestBountyFeedTTLIsBoundedByNextOpenExpiry(t *testing.T) {
	now := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		records []bountyRecord
		want    time.Duration
	}{
		{name: "empty feed", want: bountyFeedCacheTTL},
		{
			name: "soonest open bounty",
			records: []bountyRecord{
				{Status: "OPEN", ExpiresAt: now.Add(2 * time.Second)},
				{Status: "OPEN", ExpiresAt: now.Add(10 * time.Second)},
			},
			want: 2 * time.Second,
		},
		{
			name:    "closed bounty does not shorten cache",
			records: []bountyRecord{{Status: "EXPIRED", ExpiresAt: now.Add(time.Second)}},
			want:    bountyFeedCacheTTL,
		},
		{
			name:    "already expired open bounty is not cached",
			records: []bountyRecord{{Status: "OPEN", ExpiresAt: now.Add(-time.Second)}},
			want:    0,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := bountyFeedTTL(test.records, now); got != test.want {
				t.Fatalf("bountyFeedTTL() = %s, want %s", got, test.want)
			}
		})
	}
}

func TestCreateBountyIsIdempotentIntegration(t *testing.T) {
	postgresURL := os.Getenv("TEST_POSTGRES_URL")
	redisURL := os.Getenv("TEST_REDIS_URL")
	if postgresURL == "" || redisURL == "" {
		t.Skip("set TEST_POSTGRES_URL and TEST_REDIS_URL to run the persistence integration test")
	}

	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(postgres.Open(postgresURL), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&bountyRecord{}, &idempotencyRecord{}); err != nil {
		t.Fatal(err)
	}
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		t.Fatal(err)
	}
	redisClient := redis.NewClient(options)
	t.Cleanup(func() { _ = redisClient.Close() })
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		t.Fatal(err)
	}

	key := "integration-" + uuid.NewString()
	keyHash := hashString(key)
	cacheKey := "idempotency:v1:" + keyHash
	createdID := ""
	t.Cleanup(func() {
		if err := redisClient.Del(context.Background(), cacheKey).Err(); err != nil {
			t.Errorf("delete Redis test key: %v", err)
		}
		if err := db.Where("key_hash = ?", keyHash).Delete(&idempotencyRecord{}).Error; err != nil {
			t.Errorf("delete PostgreSQL test idempotency record: %v", err)
		}
		if createdID != "" {
			if err := db.Where("id = ?", createdID).Delete(&bountyRecord{}).Error; err != nil {
				t.Errorf("delete PostgreSQL test bounty: %v", err)
			}
			if err := redisClient.Incr(context.Background(), bountyFeedRevisionKey).Err(); err != nil {
				t.Errorf("invalidate test bounty feed cache: %v", err)
			}
		}
	})
	app := &api{
		db: db, redis: redisClient,
		matcher: fixedMatcher{matches: []graph.ExpertMatch{{
			ID: "engineer-1", Name: "Sample Engineer", Team: "Platform",
			Score: 1.25, MatchedSkills: []string{"go"},
		}}},
	}
	initialFeed := sendFeedRequest(app.routes())
	if initialFeed.Code != http.StatusOK {
		t.Fatalf("initial feed status = %d, body = %s", initialFeed.Code, initialFeed.Body.String())
	}
	requestBody := []byte(`{"title":"Improve API latency","description":"Example","skills":["go"],"bounty_amount":150,"ttl_seconds":3600,"poster_team":"Commerce"}`)

	first := sendCreateRequest(app.routes(), key, requestBody)
	if first.Code != http.StatusCreated {
		t.Fatalf("first request status = %d, body = %s", first.Code, first.Body.String())
	}
	var createdResult createBountyResponse
	if err := json.Unmarshal(first.Body.Bytes(), &createdResult); err != nil {
		t.Fatal(err)
	}
	created := createdResult.Bounty
	createdID = created.ID
	if created.ID == "" || len(created.MatchedExperts) != 1 {
		t.Fatalf("unexpected create response: %+v", created)
	}
	updatedFeed := sendFeedRequest(app.routes())
	if updatedFeed.Code != http.StatusOK {
		t.Fatalf("updated feed status = %d, body = %s", updatedFeed.Code, updatedFeed.Body.String())
	}
	var feedResult bountyFeedResponse
	if err := json.Unmarshal(updatedFeed.Body.Bytes(), &feedResult); err != nil {
		t.Fatal(err)
	}
	foundCreatedBounty := false
	for _, bounty := range feedResult.Bounties {
		if bounty.ID == created.ID {
			foundCreatedBounty = true
			break
		}
	}
	if !foundCreatedBounty {
		t.Fatalf("feed cache was not invalidated after creating bounty %q", created.ID)
	}
	if _, _, cacheHit := app.cachedBountyFeed(context.Background()); !cacheHit {
		t.Fatal("fresh feed response was not cached")
	}

	replay := sendCreateRequest(app.routes(), key, requestBody)
	if replay.Code != http.StatusOK {
		t.Fatalf("replay status = %d, body = %s", replay.Code, replay.Body.String())
	}
	var replayedResult createBountyResponse
	if err := json.Unmarshal(replay.Body.Bytes(), &replayedResult); err != nil {
		t.Fatal(err)
	}
	replayed := replayedResult.Bounty
	if replayed.ID != created.ID {
		t.Fatalf("replay returned bounty %q, want original %q", replayed.ID, created.ID)
	}
	if err := redisClient.Del(context.Background(), cacheKey).Err(); err != nil {
		t.Fatal(err)
	}
	durableReplay := sendCreateRequest(app.routes(), key, requestBody)
	if durableReplay.Code != http.StatusOK {
		t.Fatalf("database replay status = %d, body = %s", durableReplay.Code, durableReplay.Body.String())
	}
	var restoredResult createBountyResponse
	if err := json.Unmarshal(durableReplay.Body.Bytes(), &restoredResult); err != nil {
		t.Fatal(err)
	}
	restored := restoredResult.Bounty
	if restored.ID != created.ID {
		t.Fatalf("database replay returned bounty %q, want original %q", restored.ID, created.ID)
	}

	changedRequest := bytes.Replace(requestBody, []byte("Example"), []byte("Changed"), 1)
	conflict := sendCreateRequest(app.routes(), key, changedRequest)
	if conflict.Code != http.StatusConflict {
		t.Fatalf("reused key with different body status = %d, body = %s", conflict.Code, conflict.Body.String())
	}

	var count int64
	if err := db.Model(&bountyRecord{}).Where("id = ?", created.ID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("stored bounty count = %d, want 1", count)
	}
}

func sendCreateRequest(router http.Handler, idempotencyKey string, body []byte) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/bounties", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Idempotency-Key", idempotencyKey)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func sendFeedRequest(router http.Handler) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/bounties", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

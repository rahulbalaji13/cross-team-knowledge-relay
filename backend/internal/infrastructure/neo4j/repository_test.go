package neo4j

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	driver "github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func TestMatchExpertsFiltersTeamAndRanksSkillStrengthIntegration(t *testing.T) {
	uri := os.Getenv("TEST_NEO4J_URI")
	username := os.Getenv("TEST_NEO4J_USERNAME")
	password := os.Getenv("TEST_NEO4J_PASSWORD")
	if uri == "" || username == "" || password == "" {
		t.Skip("set TEST_NEO4J_URI, TEST_NEO4J_USERNAME, and TEST_NEO4J_PASSWORD to run the Neo4j integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	repository, err := NewRepository(ctx, uri, username, password)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := repository.Close(context.Background()); err != nil {
			t.Errorf("close Neo4j test connection: %v", err)
		}
	})
	if err := repository.EnsureSchema(ctx); err != nil {
		t.Fatal(err)
	}

	suffix := uuid.NewString()
	skillName := "test-skill-" + suffix
	teamName := "test-team-" + suffix
	posterTeam := "poster-team-" + suffix
	experts := []map[string]any{
		{"id": "test-expert-high-" + suffix, "name": "High Match", "team": teamName, "level": 5, "lastUsed": time.Now().UTC().Format("2006-01-02"), "reputation": 4.8},
		{"id": "test-expert-low-" + suffix, "name": "Low Match", "team": teamName, "level": 2, "lastUsed": "2020-01-01", "reputation": 3.0},
		{"id": "test-expert-same-team-" + suffix, "name": "Same Team", "team": posterTeam, "level": 5, "lastUsed": time.Now().UTC().Format("2006-01-02"), "reputation": 5.0},
	}
	t.Cleanup(func() {
		cleanupTestGraph(t, repository, experts, skillName, teamName, posterTeam)
	})

	session := repository.driver.NewSession(ctx, driver.SessionConfig{AccessMode: driver.AccessModeWrite})
	defer session.Close(ctx)
	result, err := session.Run(ctx, `
UNWIND $experts AS row
MERGE (expert:Engineer {id: row.id})
SET expert.name = row.name, expert.team = row.team, expert.reputation = row.reputation
MERGE (team:Team {name: row.team})
MERGE (expert)-[:BELONGS_TO]->(team)
MERGE (skill:Skill {name: $skill})
MERGE (expert)-[edge:HAS_SKILL]->(skill)
SET edge.level = row.level, edge.last_used = row.lastUsed`, map[string]any{
		"experts": experts, "skill": skillName,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := result.Consume(ctx); err != nil {
		t.Fatal(err)
	}

	matches, err := repository.MatchExperts(ctx, []string{skillName}, posterTeam, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 2 {
		t.Fatalf("got %d matches, want 2 (same-team expert should be excluded)", len(matches))
	}
	highExpertID := experts[0]["id"].(string)
	if matches[0].ID != highExpertID {
		t.Fatalf("top match = %q, want stronger and more recent expert %q", matches[0].ID, highExpertID)
	}
	for _, match := range matches {
		if match.Team == posterTeam {
			t.Fatalf("same-team expert %q was not filtered", match.ID)
		}
	}
}

func cleanupTestGraph(t *testing.T, repository *Repository, experts []map[string]any, skillName string, teams ...string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	session := repository.driver.NewSession(ctx, driver.SessionConfig{AccessMode: driver.AccessModeWrite})
	defer session.Close(ctx)
	ids := make([]string, 0, len(experts))
	for _, expert := range experts {
		ids = append(ids, expert["id"].(string))
	}
	result, err := session.Run(ctx, `
MATCH (expert:Engineer) WHERE expert.id IN $ids DETACH DELETE expert
`, map[string]any{"ids": ids})
	if err != nil {
		t.Errorf("cleanup test experts: %v", err)
		return
	}
	if _, err := result.Consume(ctx); err != nil {
		t.Errorf("cleanup test graph: %v", err)
		return
	}
	result, err = session.Run(ctx, "MATCH (skill:Skill {name: $skill}) DETACH DELETE skill", map[string]any{"skill": skillName})
	if err != nil {
		t.Errorf("cleanup test skill: %v", err)
		return
	}
	if _, err := result.Consume(ctx); err != nil {
		t.Errorf("cleanup test skill: %v", err)
	}
	for _, team := range teams {
		result, err := session.Run(ctx, `
MATCH (team:Team {name: $name})
WHERE NOT EXISTS { MATCH (:Engineer)-[:BELONGS_TO]->(team) }
DELETE team`, map[string]any{"name": team})
		if err != nil {
			t.Errorf("prepare cleanup for test team %s: %v", team, err)
			continue
		}
		if _, err := result.Consume(ctx); err != nil {
			t.Errorf("cleanup test team %s: %v", team, err)
		}
	}
}

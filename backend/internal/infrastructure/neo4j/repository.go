package neo4j

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	driver "github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type ExpertMatch struct {
	ID            string   `json:"expert_id"`
	Name          string   `json:"name"`
	Team          string   `json:"team"`
	Score         float64  `json:"score"`
	MatchedSkills []string `json:"matched_skills"`
}

type Repository struct {
	driver driver.DriverWithContext
}

func NewRepository(ctx context.Context, uri, username, password string) (*Repository, error) {
	client, err := driver.NewDriverWithContext(uri, driver.BasicAuth(username, password, ""))
	if err != nil {
		return nil, fmt.Errorf("create Neo4j driver: %w", err)
	}
	if err := client.VerifyConnectivity(ctx); err != nil {
		_ = client.Close(ctx)
		return nil, fmt.Errorf("connect to Neo4j: %w", err)
	}
	return &Repository{driver: client}, nil
}

func (r *Repository) Close(ctx context.Context) error {
	return r.driver.Close(ctx)
}

func (r *Repository) Ping(ctx context.Context) error {
	return r.driver.VerifyConnectivity(ctx)
}

func (r *Repository) EnsureSchema(ctx context.Context) error {
	session := r.driver.NewSession(ctx, driver.SessionConfig{AccessMode: driver.AccessModeWrite})
	defer session.Close(ctx)

	statements := []string{
		"CREATE CONSTRAINT engineer_id IF NOT EXISTS FOR (e:Engineer) REQUIRE e.id IS UNIQUE",
		"CREATE CONSTRAINT team_name IF NOT EXISTS FOR (t:Team) REQUIRE t.name IS UNIQUE",
		"CREATE CONSTRAINT skill_name IF NOT EXISTS FOR (s:Skill) REQUIRE s.name IS UNIQUE",
	}
	for _, statement := range statements {
		result, err := session.Run(ctx, statement, nil)
		if err != nil {
			return fmt.Errorf("ensure Neo4j schema: %w", err)
		}
		if _, err := result.Consume(ctx); err != nil {
			return fmt.Errorf("apply Neo4j schema: %w", err)
		}
	}
	return nil
}

func (r *Repository) MatchExperts(ctx context.Context, skills []string, posterTeam string, limit int) ([]ExpertMatch, error) {
	if len(skills) == 0 {
		return []ExpertMatch{}, nil
	}
	normalizedSkills := make([]string, 0, len(skills))
	for _, skill := range skills {
		normalizedSkills = append(normalizedSkills, strings.ToLower(strings.TrimSpace(skill)))
	}

	session := r.driver.NewSession(ctx, driver.SessionConfig{AccessMode: driver.AccessModeRead})
	defer session.Close(ctx)

	const query = `
UNWIND $skills AS requiredSkill
MATCH (skill:Skill {name: requiredSkill})<-[hasSkill:HAS_SKILL]-(expert:Engineer)
MATCH (expert)-[:BELONGS_TO]->(team:Team)
WHERE toLower(team.name) <> toLower($posterTeam)
WITH expert, collect(DISTINCT {
  name: skill.name,
  level: toFloat(hasSkill.level),
  daysSince: duration.inDays(date(hasSkill.last_used), date()).days
}) AS matches
WITH expert, matches,
  reduce(total = 0.0, item IN matches |
    total + (item.level / 5.0 * 0.6) + (exp(-0.03 * item.daysSince) * 0.4)
  ) + (coalesce(expert.reputation, 3.0) / 5.0) AS score
RETURN expert.id AS id, expert.name AS name, expert.team AS team,
  score AS score, [item IN matches | item.name] AS matchedSkills
ORDER BY score DESC
LIMIT $limit`

	result, err := session.Run(ctx, query, map[string]any{
		"skills": normalizedSkills, "posterTeam": posterTeam, "limit": limit,
	})
	if err != nil {
		return nil, fmt.Errorf("run Neo4j matching query: %w", err)
	}
	records, err := result.Collect(ctx)
	if err != nil {
		return nil, fmt.Errorf("read Neo4j matching results: %w", err)
	}

	matches := make([]ExpertMatch, 0, len(records))
	for _, record := range records {
		idValue, idExists := record.Get("id")
		nameValue, nameExists := record.Get("name")
		teamValue, teamExists := record.Get("team")
		scoreValue, scoreExists := record.Get("score")
		skillValues, skillsExist := record.Get("matchedSkills")
		id, idOK := idValue.(string)
		name, nameOK := nameValue.(string)
		team, teamOK := teamValue.(string)
		numericScore, scoreOK := scoreValue.(float64)
		values, skillsOK := skillValues.([]any)
		if !idExists || !nameExists || !teamExists || !scoreExists || !skillsExist ||
			!idOK || !nameOK || !teamOK || !scoreOK || !skillsOK {
			return nil, fmt.Errorf("Neo4j returned expert data with unexpected types")
		}
		skillNames := make([]string, 0, len(values))
		for _, value := range values {
			skillName, ok := value.(string)
			if !ok {
				return nil, fmt.Errorf("Neo4j returned a non-string matched skill")
			}
			skillNames = append(skillNames, skillName)
		}
		matches = append(matches, ExpertMatch{
			ID:            id,
			Name:          name,
			Team:          team,
			Score:         numericScore,
			MatchedSkills: skillNames,
		})
	}
	return matches, nil
}

func (r *Repository) SeedEngineers(ctx context.Context, engineerCount, skillsPerEngineer int) (int, error) {
	skillNames := []string{
		"go", "neo4j", "redis", "distributed systems", "kafka", "python",
		"recommendation", "frontend", "react", "next.js", "typescript",
		"postgresql", "kubernetes", "observability", "security", "graphql",
		"machine learning", "data engineering", "java", "terraform",
	}
	teams := []string{
		"Payments", "Infra", "Search", "Commerce", "Platform",
		"Developer Experience", "Data", "Security", "Mobile", "Core",
	}
	if skillsPerEngineer > len(skillNames) {
		return 0, fmt.Errorf("skills per engineer cannot exceed %d", len(skillNames))
	}

	random := rand.New(rand.NewSource(42))
	now := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
	rows := make([]map[string]any, 0, 500)
	totalEdges := 0
	session := r.driver.NewSession(ctx, driver.SessionConfig{AccessMode: driver.AccessModeWrite})
	defer session.Close(ctx)

	const writeBatch = `
UNWIND $rows AS row
MERGE (expert:Engineer {id: row.id})
SET expert.name = row.name, expert.team = row.team, expert.reputation = row.reputation
MERGE (team:Team {name: row.team})
MERGE (expert)-[:BELONGS_TO]->(team)
MERGE (skill:Skill {name: row.skill})
MERGE (expert)-[edge:HAS_SKILL]->(skill)
SET edge.level = row.level, edge.last_used = row.lastUsed`

	flush := func() error {
		if len(rows) == 0 {
			return nil
		}
		result, err := session.Run(ctx, writeBatch, map[string]any{"rows": rows})
		if err != nil {
			return fmt.Errorf("write synthetic Neo4j data: %w", err)
		}
		if _, err := result.Consume(ctx); err != nil {
			return fmt.Errorf("commit synthetic Neo4j data: %w", err)
		}
		rows = rows[:0]
		return nil
	}

	for i := 1; i <= engineerCount; i++ {
		permutation := random.Perm(len(skillNames))[:skillsPerEngineer]
		team := teams[(i-1)%len(teams)]
		reputation := 3.0 + random.Float64()*2
		for _, skillIndex := range permutation {
			lastUsed := now.AddDate(0, 0, -random.Intn(365)).Format("2006-01-02")
			rows = append(rows, map[string]any{
				"id": fmt.Sprintf("eng-%06d", i), "name": fmt.Sprintf("Engineer %06d", i),
				"team": team, "reputation": reputation, "skill": skillNames[skillIndex],
				"level": 1 + random.Intn(5), "lastUsed": lastUsed,
			})
			totalEdges++
			if len(rows) == 500 {
				if err := flush(); err != nil {
					return totalEdges, err
				}
			}
		}
	}
	if err := flush(); err != nil {
		return totalEdges, err
	}
	return totalEdges, nil
}

func (r *Repository) CountSeededEngineers(ctx context.Context, engineerCount int) (int64, int64, error) {
	ids := make([]string, engineerCount)
	for i := range ids {
		ids[i] = fmt.Sprintf("eng-%06d", i+1)
	}

	session := r.driver.NewSession(ctx, driver.SessionConfig{AccessMode: driver.AccessModeRead})
	defer session.Close(ctx)
	result, err := session.Run(ctx, `
UNWIND $ids AS id
OPTIONAL MATCH (expert:Engineer {id: id})
OPTIONAL MATCH (expert)-[edge:HAS_SKILL]->(:Skill)
RETURN count(DISTINCT expert) AS engineerCount, count(edge) AS skillEdges
`, map[string]any{"ids": ids})
	if err != nil {
		return 0, 0, fmt.Errorf("count seeded Neo4j data: %w", err)
	}
	records, err := result.Collect(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("read seeded Neo4j counts: %w", err)
	}
	if len(records) != 1 {
		return 0, 0, fmt.Errorf("Neo4j returned %d seed count rows, expected 1", len(records))
	}
	engineerValue, _ := records[0].Get("engineerCount")
	edgeValue, _ := records[0].Get("skillEdges")
	engineerTotal, engineerOK := engineerValue.(int64)
	edgeTotal, edgeOK := edgeValue.(int64)
	if !engineerOK || !edgeOK {
		return 0, 0, fmt.Errorf("Neo4j returned unexpected seed count types")
	}
	return engineerTotal, edgeTotal, nil
}

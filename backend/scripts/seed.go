package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	graph "github.com/rahulbalaji13/cross-team-knowledge-relay/backend/internal/infrastructure/neo4j"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	uri := requiredEnv("NEO4J_URI")
	username := requiredEnv("NEO4J_USERNAME")
	password := requiredEnv("NEO4J_PASSWORD")
	engineerCount := envInt("ENGINEER_COUNT", 10000)
	skillsPerEngineer := envInt("SKILLS_PER_ENGINEER", 5)

	repository, err := graph.NewRepository(ctx, uri, username, password)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := repository.Close(context.Background()); err != nil {
			log.Printf("close Neo4j connection: %v", err)
		}
	}()
	if err := repository.EnsureSchema(ctx); err != nil {
		log.Fatal(err)
	}
	edges, err := repository.SeedEngineers(ctx, engineerCount, skillsPerEngineer)
	if err != nil {
		log.Fatal(err)
	}
	storedEngineers, storedEdges, err := repository.CountSeededEngineers(ctx, engineerCount)
	if err != nil {
		log.Fatal(err)
	}
	if storedEngineers != int64(engineerCount) || storedEdges != int64(edges) {
		log.Fatalf("seed verification failed: found %d engineers and %d skill relationships, want %d and %d",
			storedEngineers, storedEdges, engineerCount, edges)
	}
	log.Printf("Verified %d deterministic synthetic engineers and %d skill relationships", storedEngineers, storedEdges)
}

func requiredEnv(name string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		log.Fatalf("%s must be set", name)
	}
	return value
}

func envInt(name string, fallback int) int {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		log.Fatalf("%s must be a positive integer", name)
	}
	return parsed
}

// Query: Find top 5 experts for a specific skill requirement, actively excluding the bounty poster's immediate team
// Calculates score based on skill level and overall reputation to enforce PageRank/Centrality analogues.

MATCH (poster:Engineer {id: $poster_id})-[:BELONGS_TO]->(posterTeam:Team)
MATCH (bounty:Bounty {id: $bounty_id})-[:REQUIRES]->(s:Skill)<-[hs:HAS_SKILL]-(expert:Engineer)

// Core Filter: Exclude anyone in the same organizational boundary
WHERE NOT (expert)-[:BELONGS_TO]->(posterTeam)

// Recency Penalty: if last_used is old, we decay the score.
// We approximate decay here or pass variables down to application layer.
WITH expert, hs.level AS skillLevel, expert.reputation AS rep, duration.inDays(datetime(hs.last_used), datetime()).days AS daysSince

// Calculate Weight: 60% Skill Level, 40% Reputation, penalized slightly by decay over time
WITH expert, (skillLevel * 0.6 + rep * 0.4) - (daysSince * 0.01) AS finalScore

ORDER BY finalScore DESC
LIMIT 5

RETURN expert.id AS expert_id, expert.name AS name, finalScore

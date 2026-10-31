# Synthetic Dataset

This folder contains a deterministic sample CSV dataset for the Cross-Team Knowledge Relay project. It is safe demo data and does not represent real employees, companies, payments, or production activity.

## Files

| File | Purpose |
|---|---|
| `teams.csv` | Engineering teams and their business/technical domains. |
| `skills.csv` | Skill catalog used by bounties and expert profiles. |
| `engineers.csv` | Synthetic engineers with team, level, GitHub-style username, and reputation score. |
| `engineer_skills.csv` | Graph-style `Engineer -> HAS_SKILL -> Skill` edges with skill level and recency. |
| `bounties.csv` | Synthetic bounty requests with poster, required skills, amount, TTL, status, and timestamps. |
| `solved_bounties.csv` | Historical `Engineer -> SOLVED -> Bounty` edges with ratings. |
| `collaborations.csv` | Historical engineer-to-engineer collaboration edges. |
| `escrows.csv` | Transaction-style escrow records for resolved bounties. |

## Dataset shape

- 6 teams
- 15 skills
- 30 engineers
- 150+ engineer-skill edges
- 40 bounties
- Historical solved-bounty, collaboration, and escrow records

## How to use in interviews

Use this dataset to explain that the project separates relationship-heavy matching data from transaction-heavy financial data:

- Neo4j-style CSVs: `engineers.csv`, `teams.csv`, `skills.csv`, `engineer_skills.csv`, `bounties.csv`, `solved_bounties.csv`, and `collaborations.csv`.
- PostgreSQL-style CSVs: `engineers.csv`, `bounties.csv`, and `escrows.csv`.

The matching engine can join required bounty skills with engineer skills, exclude engineers from the poster's own team, and rank candidates using skill level, reputation, and recency.

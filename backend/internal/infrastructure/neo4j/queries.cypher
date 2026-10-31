UNWIND [name IN $skills | toLower(name)] AS requiredSkill
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
LIMIT $limit

-- Indexes supporting SQL-side candidate selection for speaking opportunities.
--
-- Ranking used to load every community, talk, relationship and outreach row and
-- build the full community x talk cross product in memory on every request.
-- The candidate query replaces that with a topic-overlap / existing-relationship
-- filter, which needs these indexes to stay cheap.

CREATE INDEX IF NOT EXISTS talks_topics_gin_idx ON talks USING GIN(topics);

CREATE INDEX IF NOT EXISTS relationships_community_idx
  ON relationships(project_id, community_id)
  WHERE community_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS outreach_community_idx
  ON outreach(project_id, community_id)
  WHERE community_id IS NOT NULL;

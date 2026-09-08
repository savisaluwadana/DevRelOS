-- Distinguishes signals whose author is reporting a problem from signals that
-- merely announce something.
--
-- Pain-point clustering looks for friction vocabulary, and terms like "setup",
-- "configure", "complex", "manual" and "hard" occur constantly in technical
-- writing that describes rather than complains. Ingesting blog feeds and
-- release notes therefore produced pain points backed by press releases: on a
-- real dataset, a cluster of 30 "signals" was entirely CNCF announcement posts.
-- No vocabulary tuning fixes that, because "how to configure X" and
-- "configuring X is painful" contain the same words.
--
-- 'unknown' is the default so existing rows keep their current clustering
-- behaviour; only sources that explicitly declare themselves announcements are
-- excluded from pain-point evidence.
ALTER TABLE signals
  ADD COLUMN source_shape TEXT NOT NULL DEFAULT 'unknown'
  CHECK (source_shape IN ('report', 'announcement', 'unknown'));

CREATE INDEX signals_source_shape_idx ON signals(project_id, source_shape);

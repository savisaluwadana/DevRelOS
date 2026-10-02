DROP INDEX IF EXISTS signals_source_shape_idx;
ALTER TABLE signals DROP COLUMN IF EXISTS source_shape;

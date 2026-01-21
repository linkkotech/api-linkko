-- Drop triggers
DROP TRIGGER IF EXISTS "update_pipeline_stages_updatedAt" ON public."PipelineStage";
DROP TRIGGER IF EXISTS "update_pipelines_updatedAt" ON public."Pipeline";

-- Drop indexes
DROP INDEX IF EXISTS "idx_pipeline_stages_group";
DROP INDEX IF EXISTS "idx_pipeline_stages_order";
DROP INDEX IF EXISTS "idx_pipeline_stages_pipeline";
DROP INDEX IF EXISTS "idx_pipelines_owner";
DROP INDEX IF EXISTS "idx_pipelines_active";
DROP INDEX IF EXISTS "idx_pipelines_workspace";
DROP INDEX IF EXISTS "idx_unique_default_pipeline";

-- Drop tables (CASCADE to handle foreign keys)
DROP TABLE IF EXISTS public."PipelineStage" CASCADE;
DROP TABLE IF EXISTS public."Pipeline" CASCADE;

-- Drop ENUMs (commented out to avoid breaking Next.js)
-- DROP TYPE IF EXISTS public."PipelineType";
-- DROP TYPE IF EXISTS public."StageGroup";

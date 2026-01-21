-- Migration: 000006_pipelines.up.sql
-- Description: Create Pipeline and PipelineStage domains for sales funnel management
-- Date: 2026-01-20
-- IMPORTANT: Uses camelCase column names (matching Prisma schema from Next.js)

-- Defensive creation of StageGroup ENUM
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'StageGroup') THEN
        CREATE TYPE public."StageGroup" AS ENUM (
            'active',   -- Deal is in progress
            'won',      -- Deal closed successfully
            'lost'      -- Deal lost/rejected
        );
    END IF;
END$$;

-- Defensive creation of PipelineType ENUM
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'PipelineType') THEN
        CREATE TYPE public."PipelineType" AS ENUM (
            'sales',        -- Standard B2B sales pipeline
            'recruitment',  -- Hiring pipeline
            'onboarding',   -- Customer onboarding
            'custom'        -- User-defined process
        );
    END IF;
END$$;

-- Create Pipeline table (camelCase columns)
CREATE TABLE IF NOT EXISTS public."Pipeline" (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "workspaceId" UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Pipeline configuration
    "pipelineType" public."PipelineType" NOT NULL DEFAULT 'sales',
    "isActive" BOOLEAN NOT NULL DEFAULT true,
    "isDefault" BOOLEAN NOT NULL DEFAULT false,
    
    -- Ownership
    "ownerId" UUID NOT NULL,
    
    -- Timestamps
    "createdAt" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "updatedAt" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "deletedAt" TIMESTAMPTZ,
    
    -- Constraints
    CONSTRAINT "unique_pipeline_name_per_workspace" UNIQUE("workspaceId", name) 
        WHERE "deletedAt" IS NULL
);

-- Create PipelineStage table (camelCase columns)
CREATE TABLE IF NOT EXISTS public."PipelineStage" (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "pipelineId" UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Stage configuration
    "stageGroup" public."StageGroup" NOT NULL DEFAULT 'active',
    "orderIndex" INTEGER NOT NULL DEFAULT 0,  -- Integer for ordering (simpler than DECIMAL)
    probability INTEGER NOT NULL DEFAULT 0,   -- Win probability 0-100%
    
    -- Automation (optional)
    "autoArchiveAfterDays" INTEGER,
    
    -- Timestamps
    "createdAt" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "updatedAt" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "deletedAt" TIMESTAMPTZ,
    
    -- Constraints
    FOREIGN KEY ("pipelineId") REFERENCES public."Pipeline"(id) ON DELETE CASCADE,
    CONSTRAINT "unique_stage_name_per_pipeline" UNIQUE("pipelineId", name) 
        WHERE "deletedAt" IS NULL,
    CONSTRAINT "check_probability_range" CHECK (probability >= 0 AND probability <= 100)
);

-- UNIQUE PARTIAL INDEX: Only one default pipeline per workspace
-- This is the "Constitutional Law" that prevents duplicates even if code fails
CREATE UNIQUE INDEX IF NOT EXISTS "idx_unique_default_pipeline" 
    ON public."Pipeline"("workspaceId") 
    WHERE "isDefault" = true AND "deletedAt" IS NULL;

-- Pipeline indexes
CREATE INDEX IF NOT EXISTS "idx_pipelines_workspace" 
    ON public."Pipeline"("workspaceId") 
    WHERE "deletedAt" IS NULL;

CREATE INDEX IF NOT EXISTS "idx_pipelines_active" 
    ON public."Pipeline"("workspaceId", "isActive") 
    WHERE "deletedAt" IS NULL AND "isActive" = true;

CREATE INDEX IF NOT EXISTS "idx_pipelines_owner" 
    ON public."Pipeline"("ownerId") 
    WHERE "deletedAt" IS NULL;

-- PipelineStage indexes
CREATE INDEX IF NOT EXISTS "idx_pipeline_stages_pipeline" 
    ON public."PipelineStage"("pipelineId") 
    WHERE "deletedAt" IS NULL;

CREATE INDEX IF NOT EXISTS "idx_pipeline_stages_order" 
    ON public."PipelineStage"("pipelineId", "orderIndex") 
    WHERE "deletedAt" IS NULL;

CREATE INDEX IF NOT EXISTS "idx_pipeline_stages_group" 
    ON public."PipelineStage"("pipelineId", "stageGroup") 
    WHERE "deletedAt" IS NULL;

-- Triggers for updated_at (reuse existing function)
CREATE TRIGGER IF NOT EXISTS "update_pipelines_updatedAt"
    BEFORE UPDATE ON public."Pipeline"
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER IF NOT EXISTS "update_pipeline_stages_updatedAt"
    BEFORE UPDATE ON public."PipelineStage"
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

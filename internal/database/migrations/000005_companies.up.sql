-- Migration: 000005_companies.up.sql
-- Description: Create Company domain with lifecycle stages and size classification
-- Date: 2026-01-20
-- IMPORTANT: Uses camelCase column names (matching Prisma schema from Next.js)

-- Defensive creation of CompanyLifecycleStage ENUM
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'CompanyLifecycleStage') THEN
        CREATE TYPE public."CompanyLifecycleStage" AS ENUM (
            'lead',           -- Initial contact, not qualified
            'prospect',       -- Qualified lead
            'customer',       -- Active customer
            'partner',        -- Strategic partner
            'inactive',       -- Former customer/churned
            'evangelist'      -- Happy customer promoting us
        );
    END IF;
END$$;

-- Defensive creation of CompanySize ENUM
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'CompanySize') THEN
        CREATE TYPE public."CompanySize" AS ENUM (
            'solopreneur',    -- 1 person
            'small',          -- 2-10
            'medium',         -- 11-50
            'midmarket',      -- 51-200
            'enterprise',     -- 201-1000
            'large_enterprise' -- 1000+
        );
    END IF;
END$$;

-- Create Company table (camelCase columns)
CREATE TABLE IF NOT EXISTS public."Company" (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    "workspaceId" UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    domain VARCHAR(255),
    
    -- Company classification
    industry VARCHAR(100),
    "lifecycleStage" public."CompanyLifecycleStage" NOT NULL DEFAULT 'lead',
    "companySize" public."CompanySize" NOT NULL DEFAULT 'small',
    
    -- Contact information
    phone VARCHAR(50),
    email VARCHAR(255),
    website VARCHAR(500),
    
    -- Address (JSONB for flexibility)
    address JSONB DEFAULT '{}',
    
    -- Business metrics
    "annualRevenue" DOUBLE PRECISION,
    "employeeCount" INTEGER,
    
    -- Ownership
    "ownerId" UUID NOT NULL,
    
    -- Metadata
    tags TEXT[],
    "customFields" JSONB DEFAULT '{}',
    notes TEXT,
    
    -- Timestamps
    "createdAt" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "updatedAt" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "deletedAt" TIMESTAMPTZ,
    
    -- Constraints
    CONSTRAINT "unique_domain_per_workspace" UNIQUE("workspaceId", domain) 
        WHERE "deletedAt" IS NULL AND domain IS NOT NULL
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS "idx_companies_workspace" 
    ON public."Company"("workspaceId") 
    WHERE "deletedAt" IS NULL;

CREATE INDEX IF NOT EXISTS "idx_companies_owner" 
    ON public."Company"("ownerId") 
    WHERE "deletedAt" IS NULL;

CREATE INDEX IF NOT EXISTS "idx_companies_lifecycle" 
    ON public."Company"("workspaceId", "lifecycleStage") 
    WHERE "deletedAt" IS NULL;

CREATE INDEX IF NOT EXISTS "idx_companies_size" 
    ON public."Company"("workspaceId", "companySize") 
    WHERE "deletedAt" IS NULL;

CREATE INDEX IF NOT EXISTS "idx_companies_domain" 
    ON public."Company"(domain) 
    WHERE "deletedAt" IS NULL AND domain IS NOT NULL;

CREATE INDEX IF NOT EXISTS "idx_companies_search" 
    ON public."Company" 
    USING gin(to_tsvector('simple', name || ' ' || COALESCE(domain, ''))) 
    WHERE "deletedAt" IS NULL;

-- Trigger for updated_at (reuse existing function)
CREATE TRIGGER IF NOT EXISTS "update_companies_updatedAt"
    BEFORE UPDATE ON public."Company"
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

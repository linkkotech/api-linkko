-- Drop trigger
DROP TRIGGER IF EXISTS "update_companies_updatedAt" ON public."Company";

-- Drop indexes
DROP INDEX IF EXISTS "idx_companies_search";
DROP INDEX IF EXISTS "idx_companies_domain";
DROP INDEX IF EXISTS "idx_companies_size";
DROP INDEX IF EXISTS "idx_companies_lifecycle";
DROP INDEX IF EXISTS "idx_companies_owner";
DROP INDEX IF EXISTS "idx_companies_workspace";

-- Drop table
DROP TABLE IF EXISTS public."Company" CASCADE;

-- Drop ENUMs (commented out to avoid breaking Next.js)
-- DROP TYPE IF EXISTS public."CompanySize";
-- DROP TYPE IF EXISTS public."CompanyLifecycleStage";

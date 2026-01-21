-- Drop trigger
DROP TRIGGER IF EXISTS update_tasks_updated_at ON public."Task";

-- Drop indexes
DROP INDEX IF EXISTS idx_tasks_search;
DROP INDEX IF EXISTS idx_tasks_due_date;
DROP INDEX IF EXISTS idx_tasks_contact;
DROP INDEX IF EXISTS idx_tasks_owner;
DROP INDEX IF EXISTS idx_tasks_assigned_to;
DROP INDEX IF EXISTS idx_tasks_workspace_status_position;
DROP INDEX IF EXISTS idx_tasks_workspace;

-- Drop table (CASCADE to handle any future dependencies)
DROP TABLE IF EXISTS public."Task" CASCADE;

-- Drop ENUMs (only if safe - may be used by Next.js side)
-- Commented out to prevent breaking Next.js application
-- DROP TYPE IF EXISTS public."TaskType";
-- DROP TYPE IF EXISTS public."TaskStatus";
-- DROP TYPE IF EXISTS public."Priority";

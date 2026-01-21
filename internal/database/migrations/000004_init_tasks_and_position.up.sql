-- Defensive creation of ENUMs (they may already exist from Next.js Prisma)
-- PostgreSQL doesn't support CREATE TYPE IF NOT EXISTS, so we use DO blocks

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'Priority') THEN
        CREATE TYPE public."Priority" AS ENUM ('low', 'medium', 'high', 'urgent');
    END IF;
END$$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'TaskStatus') THEN
        CREATE TYPE public."TaskStatus" AS ENUM ('backlog', 'todo', 'in_progress', 'in_review', 'done', 'cancelled');
    END IF;
END$$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'TaskType') THEN
        CREATE TYPE public."TaskType" AS ENUM ('task', 'bug', 'feature', 'improvement', 'research');
    END IF;
END$$;

-- Defensive creation of Task table (may already exist from Next.js Prisma)
CREATE TABLE IF NOT EXISTS public."Task" (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    status public."TaskStatus" NOT NULL DEFAULT 'backlog',
    priority public."Priority" NOT NULL DEFAULT 'medium',
    type public."TaskType" NOT NULL DEFAULT 'task',
    position DECIMAL(20,10) NOT NULL DEFAULT 0,
    owner_id UUID NOT NULL,
    assigned_to UUID,
    contact_id UUID,
    due_date TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    
    -- Foreign key to contacts (optional relationship)
    CONSTRAINT fk_task_contact FOREIGN KEY (contact_id) 
        REFERENCES contacts(id) ON DELETE SET NULL
);

-- Ensure position column exists with correct type (core requirement for Kanban)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_schema = 'public' 
        AND table_name = 'Task' 
        AND column_name = 'position'
    ) THEN
        ALTER TABLE public."Task" ADD COLUMN position DECIMAL(20,10) NOT NULL DEFAULT 0;
    END IF;
END$$;

-- Indexes for performance and Kanban operations
CREATE INDEX IF NOT EXISTS idx_tasks_workspace ON public."Task"(workspace_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_tasks_workspace_status_position ON public."Task"(workspace_id, status, position) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_tasks_assigned_to ON public."Task"(workspace_id, assigned_to) WHERE deleted_at IS NULL AND assigned_to IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_tasks_owner ON public."Task"(owner_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_tasks_contact ON public."Task"(contact_id) WHERE deleted_at IS NULL AND contact_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_tasks_due_date ON public."Task"(workspace_id, due_date) WHERE deleted_at IS NULL AND due_date IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_tasks_search ON public."Task" USING gin(to_tsvector('simple', title || ' ' || COALESCE(description, ''))) WHERE deleted_at IS NULL;

-- Updated at trigger (reuse existing function if available)
CREATE TRIGGER IF NOT EXISTS update_tasks_updated_at
    BEFORE UPDATE ON public."Task"
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

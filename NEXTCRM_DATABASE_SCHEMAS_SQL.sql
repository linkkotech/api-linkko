-- =====================================================
-- NextCRM - Schemas SQL Completos do Banco de Dados
-- Gerado em: 20/01/2026
-- Banco: PostgreSQL com extensão pgvector
-- ORM: Prisma 6.1.0
-- =====================================================

-- =====================================================
-- EXTENSÕES
-- =====================================================

CREATE EXTENSION IF NOT EXISTS "vector";

-- =====================================================
-- 1️⃣ ENUMs (CREATE TYPE)
-- =====================================================

-- Address & Location
CREATE TYPE "AddressFinality" AS ENUM ('SHIPPING', 'BILLING', 'CORRESPONDENCE');
CREATE TYPE "AddressType" AS ENUM ('comercial', 'residencial');
CREATE TYPE "LocationMode" AS ENUM ('ONLINE', 'IN_PERSON', 'HYBRID', 'DELIVERY');

-- Company & Contact Lifecycle
CREATE TYPE "CompanySize" AS ENUM ('STARTUP', 'SMB', 'MID_MARKET', 'ENTERPRISE');
CREATE TYPE "CompanyLifecycleStage" AS ENUM ('LEAD', 'MQL', 'SQL', 'CUSTOMER', 'CHURNED');
CREATE TYPE "ContactLifecycleStage" AS ENUM ('LEAD', 'MQL', 'SQL', 'OPPORTUNITY', 'CUSTOMER', 'EVANGELIST');

-- Tags & Insights
CREATE TYPE "TagCategory" AS ENUM ('PRIORITY', 'STATUS', 'TEMPERATURE', 'TYPE', 'QUALIFICATION');
CREATE TYPE "InsightType" AS ENUM ('BEHAVIOR', 'ENGAGEMENT', 'RISK', 'OPPORTUNITY');

-- Deals & Pipeline
CREATE TYPE "DealStage" AS ENUM ('OPEN', 'WON', 'LOST');
CREATE TYPE "StageGroup" AS ENUM ('OPEN', 'ACTIVE', 'DONE', 'CLOSED');
CREATE TYPE "PipelineType" AS ENUM ('TASK', 'DEAL', 'TICKET', 'CONTACT');

-- Tasks
CREATE TYPE "TaskStatus" AS ENUM ('TODO', 'IN_PROGRESS', 'DONE', 'CANCELLED');
CREATE TYPE "TaskType" AS ENUM ('CALL', 'EMAIL', 'MEETING', 'FOLLOWUP', 'OTHER');
CREATE TYPE "Priority" AS ENUM ('LOW', 'MEDIUM', 'HIGH', 'URGENT');

-- Activities & Communication
CREATE TYPE "ActivityType" AS ENUM ('NOTE', 'TASK', 'EMAIL', 'CALL', 'MEETING', 'MESSAGE', 'LIFECYCLE_CHANGE');
CREATE TYPE "MessageDirection" AS ENUM ('INBOUND', 'OUTBOUND');
CREATE TYPE "MessageStatus" AS ENUM ('SENT', 'DELIVERED', 'READ', 'FAILED');
CREATE TYPE "EmailStatus" AS ENUM ('DRAFT', 'SENT', 'DELIVERED', 'OPENED', 'CLICKED', 'BOUNCED');
CREATE TYPE "MeetingType" AS ENUM ('CALL', 'VIDEO', 'IN_PERSON');
CREATE TYPE "AttendeeStatus" AS ENUM ('INVITED', 'ACCEPTED', 'DECLINED', 'TENTATIVE', 'ATTENDED', 'NO_SHOW');
CREATE TYPE "CommunicationChannel" AS ENUM ('WHATSAPP', 'EMAIL', 'SMS', 'CALL');

-- Followups
CREATE TYPE "FollowupStatus" AS ENUM ('PENDING', 'COMPLETED', 'CANCELLED');
CREATE TYPE "FollowupPlanStatus" AS ENUM ('ACTIVE', 'PAUSED', 'COMPLETED', 'CANCELED');
CREATE TYPE "FollowupStepStatus" AS ENUM ('PENDING', 'DRAFTED', 'WAITING_APPROVAL', 'SENT', 'SKIPPED', 'FAILED');

-- Templates & Portfolio
CREATE TYPE "TemplateType" AS ENUM ('profile_template', 'content_block');
CREATE TYPE "BlockType" AS ENUM ('HERO', 'BIO', 'LINKS', 'CONTACT', 'TEXT', 'IMAGE', 'VIDEO', 'SEPARATOR', 'SOCIAL', 'SECTION');
CREATE TYPE "PortfolioCategoryEnum" AS ENUM ('PRODUCT', 'SERVICE', 'REAL_ESTATE', 'LODGING', 'EVENT');
CREATE TYPE "PortfolioVertical" AS ENUM ('GENERAL', 'HEALTHCARE', 'AESTHETICS', 'BEAUTY', 'RETAIL', 'REAL_ESTATE', 'HOSTING', 'EVENTS', 'GENERAL_LOCAL', 'B2B_CORPORATE');
CREATE TYPE "PortfolioStatus" AS ENUM ('DRAFT', 'ACTIVE', 'INACTIVE', 'UNAVAILABLE', 'ARCHIVED');
CREATE TYPE "PortfolioVisibility" AS ENUM ('PUBLIC', 'INTERNAL');
CREATE TYPE "PriceType" AS ENUM ('FIXED', 'FROM', 'QUOTE', 'PER_UNIT', 'PER_HOUR', 'PER_DAY', 'PER_PERSON');
CREATE TYPE "AvailabilityMode" AS ENUM ('ALWAYS', 'SCHEDULE', 'STOCK', 'CALENDAR', 'SEASONAL');

-- AI & Knowledge Base
CREATE TYPE "KnowledgeSourceType" AS ENUM ('FILE', 'URL', 'YOUTUBE', 'TEXT', 'SITE_CRAWL', 'GOOGLE_DRIVE', 'NOTION');
CREATE TYPE "IndexingStatus" AS ENUM ('PENDING', 'INDEXED', 'ERROR');
CREATE TYPE "KnowledgeSearchStrategy" AS ENUM ('CONTENT', 'TAGS');
CREATE TYPE "ToolType" AS ENUM ('HTTP_REQUEST', 'SEARCH_WEB', 'CALCULATOR', 'CODE_INTERPRETER', 'EMAIL', 'CALENDAR', 'DATABASE', 'FILE_PROCESSOR', 'CUSTOM');

-- Subscription & Billing
CREATE TYPE "BillingCycle" AS ENUM ('Monthly', 'Semiannual', 'Yearly', 'OneTime', 'MONTHLY', 'ANNUAL', 'LIFETIME', 'SEMIANNUAL', 'ONETIME');
CREATE TYPE "SubscriptionStatus" AS ENUM ('Active', 'Paused', 'Cancelled', 'Expired');
CREATE TYPE "SubscriptionType" AS ENUM ('Recorrente', 'Contrato');
CREATE TYPE "ContractType" AS ENUM ('INVOICE', 'AUTOMATIC', 'AUTOMATIC_FLEXIBLE');
CREATE TYPE "BillingType" AS ENUM ('SINGLE', 'RECURRING', 'RECURRING_FLEXIBLE', 'RECURRING_INVOICED');

-- Others
CREATE TYPE "OperationMode" AS ENUM ('Comercial', 'RedeApoio', 'Hibrido');
CREATE TYPE "AttachableType" AS ENUM ('COMPANY', 'CONTACT', 'DEAL', 'EMAIL', 'MEETING', 'NOTE');

-- =====================================================
-- 2️⃣ TABELAS PRINCIPAIS (CREATE TABLE)
-- =====================================================

-- -----------------------------------------------------
-- USUÁRIOS E AUTENTICAÇÃO
-- -----------------------------------------------------

CREATE TABLE "User" (
    "id" TEXT NOT NULL,
    "supabase_user_id" TEXT,
    "name" TEXT,
    "email" TEXT,
    "emailVerified" TIMESTAMP(3),
    "image" TEXT,
    "deletedAt" TIMESTAMP(3),
    "adminRoleId" TEXT,
    "preferences" JSONB,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,
    "cargo" TEXT,
    "celular" TEXT,

    CONSTRAINT "User_pkey" PRIMARY KEY ("id")
);

CREATE TABLE "AdminRole" (
    "id" TEXT NOT NULL,
    "name" TEXT NOT NULL,

    CONSTRAINT "AdminRole_pkey" PRIMARY KEY ("id")
);

CREATE TABLE "WorkspaceRole" (
    "id" TEXT NOT NULL,
    "name" TEXT NOT NULL,

    CONSTRAINT "WorkspaceRole_pkey" PRIMARY KEY ("id")
);

-- -----------------------------------------------------
-- ORGANIZAÇÃO E WORKSPACES (MULTI-TENANT)
-- -----------------------------------------------------

CREATE TABLE "Organization" (
    "id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "document" TEXT,
    "billingAddressId" TEXT,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "Organization_pkey" PRIMARY KEY ("id")
);

CREATE TABLE "Workspace" (
    "id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "slug" TEXT NOT NULL,
    "ownerId" TEXT NOT NULL,
    "organizationId" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,
    "stripe_customer_id" TEXT,
    "stripe_subscription_id" TEXT,
    "stripe_price_id" TEXT,
    "stripe_current_period_end" TIMESTAMP(3),
    "geocodingUsage" INTEGER NOT NULL DEFAULT 0,

    CONSTRAINT "Workspace_pkey" PRIMARY KEY ("id")
);

CREATE TABLE "WorkspaceMember" (
    "id" TEXT NOT NULL,
    "userId" TEXT NOT NULL,
    "workspaceId" TEXT NOT NULL,
    "workspaceRoleId" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "WorkspaceMember_pkey" PRIMARY KEY ("id")
);

-- -----------------------------------------------------
-- CONTACTS (LEADS/CUSTOMERS)
-- -----------------------------------------------------

CREATE TABLE "Contact" (
    "id" TEXT NOT NULL,
    "fullName" TEXT NOT NULL,
    "workspaceId" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,
    "email" TEXT,
    "notes" TEXT,
    "phone" TEXT,
    
    -- Identification
    "firstName" TEXT,
    "lastName" TEXT,
    "image" TEXT,
    "whatsapp" TEXT,
    "linkedinUrl" TEXT,
    
    -- Location & Preferences
    "language" TEXT,
    "timezone" TEXT,
    "city" TEXT,
    "state" TEXT,
    "country" TEXT,
    
    -- Professional Qualification
    "jobTitle" TEXT,
    "department" TEXT,
    "decisionRole" TEXT,
    
    -- System & Management
    "tagLabels" TEXT[],
    "source" TEXT DEFAULT 'manual',
    "lastInteractionAt" TIMESTAMP(3),
    "ownerId" TEXT,
    "socialUrls" JSONB,
    
    -- Company Relationship
    "companyId" TEXT,
    
    -- CRM Enterprise Fields
    "deletedAt" TIMESTAMP(3),
    "deletedById" TEXT,
    "contactScore" INTEGER NOT NULL DEFAULT 0,
    "lifecycleStage" "ContactLifecycleStage" NOT NULL DEFAULT 'LEAD',
    "assignedToId" TEXT,
    "createdById" TEXT,
    "updatedById" TEXT,

    CONSTRAINT "Contact_pkey" PRIMARY KEY ("id")
);

-- -----------------------------------------------------
-- COMPANIES
-- -----------------------------------------------------

CREATE TABLE "Company" (
    "id" TEXT NOT NULL,
    "workspaceId" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "website" TEXT,
    "linkedin" TEXT,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,
    
    -- Legal & Contact Data
    "legalName" TEXT,
    "phone" TEXT,
    "instagram" TEXT,
    "policyUrl" TEXT,
    "socialUrls" JSONB,
    
    -- Location & Address
    "addressLine" TEXT,
    "city" TEXT,
    "state" TEXT,
    "country" TEXT,
    "timezone" TEXT,
    
    -- Regional Settings
    "currency" TEXT,
    "locale" TEXT,
    
    -- Business Hours (Structured)
    "businessHours" JSONB,
    "supportHours" JSONB,
    
    -- CRM Enterprise Fields
    "deletedAt" TIMESTAMP(3),
    "deletedById" TEXT,
    "size" "CompanySize",
    "revenue" DOUBLE PRECISION,
    "companyScore" INTEGER NOT NULL DEFAULT 0,
    "lifecycleStage" "CompanyLifecycleStage" NOT NULL DEFAULT 'LEAD',
    "assignedToId" TEXT,
    "createdById" TEXT,
    "updatedById" TEXT,

    CONSTRAINT "Company_pkey" PRIMARY KEY ("id")
);

-- -----------------------------------------------------
-- PIPELINE & STAGES
-- -----------------------------------------------------

CREATE TABLE "Pipeline" (
    "id" TEXT NOT NULL,
    "workspaceId" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "isDefault" BOOLEAN NOT NULL DEFAULT false,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "Pipeline_pkey" PRIMARY KEY ("id")
);

CREATE TABLE "PipelineStage" (
    "id" TEXT NOT NULL,
    "pipelineId" TEXT,
    "workspaceId" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "orderIndex" INTEGER NOT NULL,
    "color" TEXT,
    "group" "StageGroup" NOT NULL DEFAULT 'OPEN',
    "type" "PipelineType" NOT NULL DEFAULT 'DEAL',
    "isLocked" BOOLEAN NOT NULL DEFAULT false,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "PipelineStage_pkey" PRIMARY KEY ("id")
);

-- -----------------------------------------------------
-- DEALS
-- -----------------------------------------------------

CREATE TABLE "Deal" (
    "id" TEXT NOT NULL,
    "workspaceId" TEXT NOT NULL,
    "pipelineId" TEXT NOT NULL,
    "stageId" TEXT,
    "contactId" TEXT,
    "name" TEXT NOT NULL,
    "value" DOUBLE PRECISION,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,
    
    -- CRM Enterprise Fields
    "deletedAt" TIMESTAMP(3),
    "deletedById" TEXT,
    "description" TEXT,
    "currency" TEXT NOT NULL DEFAULT 'BRL',
    "stage" "DealStage" NOT NULL DEFAULT 'OPEN',
    "probability" INTEGER DEFAULT 50,
    "expectedCloseDate" TIMESTAMP(3),
    "closedAt" TIMESTAMP(3),
    "lostReason" TEXT,
    "companyId" TEXT,
    "ownerId" TEXT,
    "createdById" TEXT NOT NULL,
    "updatedById" TEXT,

    CONSTRAINT "Deal_pkey" PRIMARY KEY ("id")
);

-- -----------------------------------------------------
-- TASKS
-- -----------------------------------------------------

CREATE TABLE "Task" (
    "id" TEXT NOT NULL,
    "title" TEXT NOT NULL,
    "workspaceId" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,
    
    -- CRM Enterprise Fields
    "description" TEXT,
    "status" "TaskStatus" NOT NULL DEFAULT 'TODO',
    "priority" "Priority" NOT NULL DEFAULT 'MEDIUM',
    "type" "TaskType" DEFAULT 'OTHER',
    "taskType" TEXT NOT NULL DEFAULT 'TASK',
    "reminderType" TEXT NOT NULL DEFAULT 'NONE',
    "dueDate" TIMESTAMP(3),
    "completedAt" TIMESTAMP(3),
    "deletedAt" TIMESTAMP(3),
    "companyId" TEXT,
    "contactId" TEXT,
    "dealId" TEXT,
    "assignedToId" TEXT,
    "stageId" TEXT,

    CONSTRAINT "Task_pkey" PRIMARY KEY ("id")
);

-- -----------------------------------------------------
-- TASK COMMENTS (Sistema de Comentários 2.0)
-- -----------------------------------------------------

CREATE TABLE "TaskComment" (
    "id" TEXT NOT NULL,
    "taskId" TEXT NOT NULL,
    "content" TEXT NOT NULL,
    "userId" TEXT NOT NULL,
    "workspaceId" TEXT NOT NULL,
    "userName" TEXT NOT NULL,
    "userImage" TEXT,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,
    "deletedAt" TIMESTAMP(3),
    "editedAt" TIMESTAMP(3),
    
    -- Thread support (auto-relação)
    "parentId" TEXT,
    
    -- Mentions e Reactions
    "mentions" TEXT[] DEFAULT ARRAY[]::TEXT[],
    "likes" TEXT[] DEFAULT ARRAY[]::TEXT[],
    "dislikes" TEXT[] DEFAULT ARRAY[]::TEXT[],

    CONSTRAINT "TaskComment_pkey" PRIMARY KEY ("id")
);

-- -----------------------------------------------------
-- ACTIVITIES (TIMELINE)
-- -----------------------------------------------------

CREATE TABLE "Activity" (
    "id" TEXT NOT NULL,
    "workspaceId" TEXT NOT NULL,
    "companyId" TEXT,
    "contactId" TEXT,
    "dealId" TEXT,
    "activityType" "ActivityType" NOT NULL,
    "activityId" TEXT,
    "userId" TEXT NOT NULL,
    "metadata" JSONB,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "Activity_pkey" PRIMARY KEY ("id")
);

-- -----------------------------------------------------
-- NOTES
-- -----------------------------------------------------

CREATE TABLE "Note" (
    "id" TEXT NOT NULL,
    "workspaceId" TEXT NOT NULL,
    "companyId" TEXT,
    "contactId" TEXT,
    "dealId" TEXT,
    "content" TEXT NOT NULL,
    "isPinned" BOOLEAN NOT NULL DEFAULT false,
    "userId" TEXT NOT NULL,
    "deletedAt" TIMESTAMP(3),
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "Note_pkey" PRIMARY KEY ("id")
);

-- -----------------------------------------------------
-- MESSAGES
-- -----------------------------------------------------

CREATE TABLE "Message" (
    "id" TEXT NOT NULL,
    "workspaceId" TEXT NOT NULL,
    "contactId" TEXT NOT NULL,
    "companyId" TEXT,
    "direction" "MessageDirection" NOT NULL,
    "platform" TEXT NOT NULL,
    "content" TEXT NOT NULL,
    "status" "MessageStatus" NOT NULL,
    "sentAt" TIMESTAMP(3) NOT NULL,
    "userId" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "Message_pkey" PRIMARY KEY ("id")
);

-- -----------------------------------------------------
-- EMAILS
-- -----------------------------------------------------

CREATE TABLE "Email" (
    "id" TEXT NOT NULL,
    "workspaceId" TEXT NOT NULL,
    "contactId" TEXT NOT NULL,
    "companyId" TEXT,
    "subject" TEXT NOT NULL,
    "body" TEXT NOT NULL,
    "fromEmail" TEXT NOT NULL,
    "toEmail" TEXT NOT NULL,
    "ccEmails" TEXT[],
    "bccEmails" TEXT[],
    "status" "EmailStatus" NOT NULL DEFAULT 'DRAFT',
    "sentAt" TIMESTAMP(3),
    "openedAt" TIMESTAMP(3),
    "clickedAt" TIMESTAMP(3),
    "userId" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "Email_pkey" PRIMARY KEY ("id")
);

-- -----------------------------------------------------
-- CALLS
-- -----------------------------------------------------

CREATE TABLE "Call" (
    "id" TEXT NOT NULL,
    "workspaceId" TEXT NOT NULL,
    "contactId" TEXT NOT NULL,
    "companyId" TEXT,
    "direction" "MessageDirection" NOT NULL,
    "duration" INTEGER,
    "recordingUrl" TEXT,
    "summary" TEXT,
    "userId" TEXT NOT NULL,
    "calledAt" TIMESTAMP(3) NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "Call_pkey" PRIMARY KEY ("id")
);

-- -----------------------------------------------------
-- MEETINGS
-- -----------------------------------------------------

CREATE TABLE "Meeting" (
    "id" TEXT NOT NULL,
    "workspaceId" TEXT NOT NULL,
    "title" TEXT NOT NULL,
    "description" TEXT,
    "meetingType" "MeetingType" NOT NULL,
    "startTime" TIMESTAMP(3) NOT NULL,
    "endTime" TIMESTAMP(3) NOT NULL,
    "location" TEXT,
    "meetingUrl" TEXT,
    "externalId" TEXT,
    "userId" TEXT NOT NULL,
    "deletedAt" TIMESTAMP(3),
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "Meeting_pkey" PRIMARY KEY ("id")
);

CREATE TABLE "MeetingAttendee" (
    "id" TEXT NOT NULL,
    "meetingId" TEXT NOT NULL,
    "contactId" TEXT,
    "email" TEXT NOT NULL,
    "name" TEXT,
    "status" "AttendeeStatus" NOT NULL DEFAULT 'INVITED',
    "respondedAt" TIMESTAMP(3),
    "externalAttendeeId" TEXT,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "MeetingAttendee_pkey" PRIMARY KEY ("id")
);

-- -----------------------------------------------------
-- TAGS
-- -----------------------------------------------------

CREATE TABLE "Tag" (
    "id" TEXT NOT NULL,
    "workspaceId" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "category" "TagCategory" NOT NULL,
    "color" TEXT NOT NULL,
    "description" TEXT,
    "isSystem" BOOLEAN NOT NULL DEFAULT false,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "Tag_pkey" PRIMARY KEY ("id")
);

CREATE TABLE "CompanyTag" (
    "id" TEXT NOT NULL,
    "companyId" TEXT NOT NULL,
    "tagId" TEXT NOT NULL,
    "userId" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "CompanyTag_pkey" PRIMARY KEY ("id")
);

CREATE TABLE "ContactTag" (
    "id" TEXT NOT NULL,
    "contactId" TEXT NOT NULL,
    "tagId" TEXT NOT NULL,
    "userId" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "ContactTag_pkey" PRIMARY KEY ("id")
);

CREATE TABLE "DealTag" (
    "id" TEXT NOT NULL,
    "dealId" TEXT NOT NULL,
    "tagId" TEXT NOT NULL,
    "userId" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "DealTag_pkey" PRIMARY KEY ("id")
);

-- -----------------------------------------------------
-- LIFECYCLE HISTORY
-- -----------------------------------------------------

CREATE TABLE "CompanyLifecycleHistory" (
    "id" TEXT NOT NULL,
    "workspaceId" TEXT NOT NULL,
    "companyId" TEXT NOT NULL,
    "fromStage" "CompanyLifecycleStage" NOT NULL,
    "toStage" "CompanyLifecycleStage" NOT NULL,
    "reason" TEXT,
    "userId" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "CompanyLifecycleHistory_pkey" PRIMARY KEY ("id")
);

CREATE TABLE "ContactLifecycleHistory" (
    "id" TEXT NOT NULL,
    "workspaceId" TEXT NOT NULL,
    "contactId" TEXT NOT NULL,
    "fromStage" "ContactLifecycleStage" NOT NULL,
    "toStage" "ContactLifecycleStage" NOT NULL,
    "reason" TEXT,
    "userId" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "ContactLifecycleHistory_pkey" PRIMARY KEY ("id")
);

CREATE TABLE "DealStageHistory" (
    "id" TEXT NOT NULL,
    "workspaceId" TEXT NOT NULL,
    "dealId" TEXT NOT NULL,
    "fromStage" "DealStage" NOT NULL,
    "toStage" "DealStage" NOT NULL,
    "reason" TEXT,
    "userId" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "DealStageHistory_pkey" PRIMARY KEY ("id")
);

-- -----------------------------------------------------
-- INSIGHTS
-- -----------------------------------------------------

CREATE TABLE "CompanyInsight" (
    "id" TEXT NOT NULL,
    "workspaceId" TEXT NOT NULL,
    "companyId" TEXT NOT NULL,
    "insightType" "InsightType" NOT NULL,
    "title" TEXT NOT NULL,
    "description" TEXT NOT NULL,
    "metadata" JSONB,
    "userId" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "originStepId" TEXT,

    CONSTRAINT "CompanyInsight_pkey" PRIMARY KEY ("id")
);

CREATE TABLE "ContactInsight" (
    "id" TEXT NOT NULL,
    "workspaceId" TEXT NOT NULL,
    "contactId" TEXT NOT NULL,
    "insightType" "InsightType" NOT NULL,
    "title" TEXT NOT NULL,
    "description" TEXT NOT NULL,
    "metadata" JSONB,
    "userId" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "originStepId" TEXT,

    CONSTRAINT "ContactInsight_pkey" PRIMARY KEY ("id")
);

-- -----------------------------------------------------
-- FOLLOWUP PLANS
-- -----------------------------------------------------

CREATE TABLE "followup_plans" (
    "id" TEXT NOT NULL,
    "workspaceId" TEXT NOT NULL,
    "contactId" TEXT,
    "companyId" TEXT,
    "status" "FollowupPlanStatus" NOT NULL DEFAULT 'ACTIVE',
    "objective" TEXT,
    "sourceContactInsightId" TEXT,
    "sourceCompanyInsightId" TEXT,
    "journeySummarySnapshot" TEXT,
    "metrics" JSONB,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "followup_plans_pkey" PRIMARY KEY ("id")
);

CREATE TABLE "followup_steps" (
    "id" TEXT NOT NULL,
    "planId" TEXT NOT NULL,
    "workspaceId" TEXT NOT NULL,
    "channel" "CommunicationChannel" NOT NULL,
    "dayOffset" INTEGER NOT NULL,
    "timeOfDay" TEXT,
    "status" "FollowupStepStatus" NOT NULL DEFAULT 'PENDING',
    "messageTemplate" TEXT NOT NULL,
    "messageContent" TEXT,
    "sentAt" TIMESTAMP(3),
    "requiresApproval" BOOLEAN NOT NULL DEFAULT false,
    "approvedBy" TEXT,
    "approvedAt" TIMESTAMP(3),
    "systemMessageId" TEXT,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "followup_steps_pkey" PRIMARY KEY ("id")
);

CREATE TABLE "DealFollowup" (
    "id" TEXT NOT NULL,
    "workspaceId" TEXT NOT NULL,
    "dealId" TEXT NOT NULL,
    "title" TEXT NOT NULL,
    "description" TEXT,
    "dueDate" TIMESTAMP(3) NOT NULL,
    "status" "FollowupStatus" NOT NULL DEFAULT 'PENDING',
    "completedAt" TIMESTAMP(3),
    "userId" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "DealFollowup_pkey" PRIMARY KEY ("id")
);

-- -----------------------------------------------------
-- ADDRESSES
-- -----------------------------------------------------

CREATE TABLE "Address" (
    "id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "finality" "AddressFinality" NOT NULL DEFAULT 'SHIPPING',
    "recipientName" TEXT,
    "phoneNumber" TEXT,
    "street" TEXT NOT NULL,
    "number" TEXT NOT NULL,
    "complement" TEXT,
    "district" TEXT NOT NULL,
    "city" TEXT NOT NULL,
    "stateCode" TEXT NOT NULL,
    "country" TEXT NOT NULL,
    "postalCode" TEXT NOT NULL,
    "latitude" DOUBLE PRECISION,
    "longitude" DOUBLE PRECISION,
    "type" "AddressType" NOT NULL,
    "isActive" BOOLEAN NOT NULL DEFAULT true,
    "workspaceId" TEXT NOT NULL,
    "createdById" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,
    "deletedAt" TIMESTAMP(3),
    "contactId" TEXT,
    "companyId" TEXT,
    "orderIndex" INTEGER NOT NULL DEFAULT 0,

    CONSTRAINT "Address_pkey" PRIMARY KEY ("id")
);

-- -----------------------------------------------------
-- IDEMPOTENCY & AUDIT
-- -----------------------------------------------------

CREATE TABLE "idempotency_keys" (
    "id" TEXT NOT NULL,
    "key" TEXT NOT NULL,
    "workspaceId" TEXT NOT NULL,
    "response" JSONB,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "expiresAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "idempotency_keys_pkey" PRIMARY KEY ("id")
);

CREATE TABLE "audit_log" (
    "id" TEXT NOT NULL,
    "workspaceId" TEXT NOT NULL,
    "userId" TEXT NOT NULL,
    "action" TEXT NOT NULL,
    "entityType" TEXT NOT NULL,
    "entityId" TEXT NOT NULL,
    "metadata" JSONB,
    "ipAddress" TEXT,
    "userAgent" TEXT,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "audit_log_pkey" PRIMARY KEY ("id")
);

-- =====================================================
-- 3️⃣ ÍNDICES E CONSTRAINTS
-- =====================================================

-- User
CREATE UNIQUE INDEX "User_supabase_user_id_key" ON "User"("supabase_user_id");
CREATE UNIQUE INDEX "User_email_key" ON "User"("email");

-- AdminRole
CREATE UNIQUE INDEX "AdminRole_name_key" ON "AdminRole"("name");

-- WorkspaceRole
CREATE UNIQUE INDEX "WorkspaceRole_name_key" ON "WorkspaceRole"("name");

-- Organization
CREATE UNIQUE INDEX "Organization_document_key" ON "Organization"("document");
CREATE INDEX "Organization_document_idx" ON "Organization"("document");

-- Workspace
CREATE UNIQUE INDEX "Workspace_slug_key" ON "Workspace"("slug");
CREATE UNIQUE INDEX "Workspace_stripe_customer_id_key" ON "Workspace"("stripe_customer_id");
CREATE UNIQUE INDEX "Workspace_stripe_subscription_id_key" ON "Workspace"("stripe_subscription_id");
CREATE INDEX "Workspace_organizationId_idx" ON "Workspace"("organizationId");

-- WorkspaceMember
CREATE UNIQUE INDEX "WorkspaceMember_userId_workspaceId_key" ON "WorkspaceMember"("userId", "workspaceId");

-- Contact
CREATE INDEX "Contact_workspaceId_idx" ON "Contact"("workspaceId");
CREATE INDEX "Contact_email_idx" ON "Contact"("email");
CREATE INDEX "Contact_companyId_idx" ON "Contact"("companyId");
CREATE INDEX "Contact_ownerId_idx" ON "Contact"("ownerId");
CREATE INDEX "Contact_workspaceId_lifecycleStage_idx" ON "Contact"("workspaceId", "lifecycleStage");
CREATE INDEX "Contact_workspaceId_contactScore_idx" ON "Contact"("workspaceId", "contactScore" DESC);
CREATE INDEX "Contact_assignedToId_idx" ON "Contact"("assignedToId");
CREATE INDEX "Contact_deletedAt_idx" ON "Contact"("deletedAt");
CREATE INDEX "Contact_source_idx" ON "Contact"("source");
CREATE INDEX "Contact_lastInteractionAt_idx" ON "Contact"("lastInteractionAt" DESC);

-- Company
CREATE INDEX "Company_workspaceId_idx" ON "Company"("workspaceId");
CREATE INDEX "Company_workspaceId_lifecycleStage_idx" ON "Company"("workspaceId", "lifecycleStage");
CREATE INDEX "Company_workspaceId_companyScore_idx" ON "Company"("workspaceId", "companyScore" DESC);
CREATE INDEX "Company_assignedToId_idx" ON "Company"("assignedToId");
CREATE INDEX "Company_deletedAt_idx" ON "Company"("deletedAt");
CREATE INDEX "Company_createdById_idx" ON "Company"("createdById");

-- Pipeline
CREATE INDEX "Pipeline_workspaceId_idx" ON "Pipeline"("workspaceId");
CREATE INDEX "Pipeline_workspaceId_isDefault_idx" ON "Pipeline"("workspaceId", "isDefault");

-- PipelineStage
CREATE INDEX "PipelineStage_pipelineId_idx" ON "PipelineStage"("pipelineId");
CREATE INDEX "PipelineStage_workspaceId_idx" ON "PipelineStage"("workspaceId");
CREATE INDEX "PipelineStage_pipelineId_orderIndex_idx" ON "PipelineStage"("pipelineId", "orderIndex");
CREATE INDEX "PipelineStage_workspaceId_type_idx" ON "PipelineStage"("workspaceId", "type");

-- Deal
CREATE INDEX "Deal_workspaceId_idx" ON "Deal"("workspaceId");
CREATE INDEX "Deal_pipelineId_idx" ON "Deal"("pipelineId");
CREATE INDEX "Deal_stageId_idx" ON "Deal"("stageId");
CREATE INDEX "Deal_contactId_idx" ON "Deal"("contactId");
CREATE INDEX "Deal_workspaceId_stage_idx" ON "Deal"("workspaceId", "stage");
CREATE INDEX "Deal_companyId_idx" ON "Deal"("companyId");
CREATE INDEX "Deal_ownerId_idx" ON "Deal"("ownerId");
CREATE INDEX "Deal_expectedCloseDate_idx" ON "Deal"("expectedCloseDate");
CREATE INDEX "Deal_deletedAt_idx" ON "Deal"("deletedAt");
CREATE INDEX "Deal_closedAt_idx" ON "Deal"("closedAt");

-- Task
CREATE INDEX "Task_workspaceId_idx" ON "Task"("workspaceId");
CREATE INDEX "Task_companyId_idx" ON "Task"("companyId");
CREATE INDEX "Task_contactId_idx" ON "Task"("contactId");
CREATE INDEX "Task_dealId_idx" ON "Task"("dealId");
CREATE INDEX "Task_assignedToId_idx" ON "Task"("assignedToId");
CREATE INDEX "Task_stageId_idx" ON "Task"("stageId");
CREATE INDEX "Task_deletedAt_idx" ON "Task"("deletedAt");

-- TaskComment
CREATE INDEX "TaskComment_taskId_workspaceId_deletedAt_idx" ON "TaskComment"("taskId", "workspaceId", "deletedAt");
CREATE INDEX "TaskComment_userId_idx" ON "TaskComment"("userId");
CREATE INDEX "TaskComment_parentId_idx" ON "TaskComment"("parentId");

-- Activity
CREATE INDEX "Activity_contactId_createdAt_idx" ON "Activity"("contactId", "createdAt" DESC);
CREATE INDEX "Activity_companyId_activityType_idx" ON "Activity"("companyId", "activityType");
CREATE INDEX "Activity_dealId_createdAt_idx" ON "Activity"("dealId", "createdAt" DESC);
CREATE INDEX "Activity_workspaceId_activityType_idx" ON "Activity"("workspaceId", "activityType");
CREATE INDEX "Activity_userId_idx" ON "Activity"("userId");
CREATE INDEX "Activity_activityType_activityId_idx" ON "Activity"("activityType", "activityId");

-- Note
CREATE INDEX "Note_workspaceId_idx" ON "Note"("workspaceId");
CREATE INDEX "Note_contactId_idx" ON "Note"("contactId");
CREATE INDEX "Note_companyId_idx" ON "Note"("companyId");
CREATE INDEX "Note_dealId_idx" ON "Note"("dealId");
CREATE INDEX "Note_deletedAt_idx" ON "Note"("deletedAt");

-- Message
CREATE INDEX "Message_workspaceId_idx" ON "Message"("workspaceId");
CREATE INDEX "Message_contactId_idx" ON "Message"("contactId");
CREATE INDEX "Message_companyId_idx" ON "Message"("companyId");
CREATE INDEX "Message_sentAt_idx" ON "Message"("sentAt" DESC);

-- Email
CREATE INDEX "Email_workspaceId_idx" ON "Email"("workspaceId");
CREATE INDEX "Email_contactId_idx" ON "Email"("contactId");
CREATE INDEX "Email_companyId_idx" ON "Email"("companyId");
CREATE INDEX "Email_sentAt_idx" ON "Email"("sentAt" DESC);
CREATE INDEX "Email_status_idx" ON "Email"("status");

-- Call
CREATE INDEX "Call_workspaceId_idx" ON "Call"("workspaceId");
CREATE INDEX "Call_contactId_idx" ON "Call"("contactId");
CREATE INDEX "Call_companyId_idx" ON "Call"("companyId");
CREATE INDEX "Call_calledAt_idx" ON "Call"("calledAt" DESC);

-- Meeting
CREATE INDEX "Meeting_workspaceId_idx" ON "Meeting"("workspaceId");
CREATE INDEX "Meeting_userId_idx" ON "Meeting"("userId");
CREATE INDEX "Meeting_startTime_idx" ON "Meeting"("startTime");
CREATE INDEX "Meeting_deletedAt_idx" ON "Meeting"("deletedAt");
CREATE UNIQUE INDEX "Meeting_workspaceId_externalId_key" ON "Meeting"("workspaceId", "externalId");

-- MeetingAttendee
CREATE INDEX "MeetingAttendee_meetingId_idx" ON "MeetingAttendee"("meetingId");
CREATE INDEX "MeetingAttendee_contactId_idx" ON "MeetingAttendee"("contactId");
CREATE INDEX "MeetingAttendee_email_idx" ON "MeetingAttendee"("email");
CREATE UNIQUE INDEX "MeetingAttendee_meetingId_email_key" ON "MeetingAttendee"("meetingId", "email");

-- Tag
CREATE UNIQUE INDEX "Tag_workspaceId_name_key" ON "Tag"("workspaceId", "name");
CREATE INDEX "Tag_workspaceId_category_idx" ON "Tag"("workspaceId", "category");
CREATE INDEX "Tag_isSystem_idx" ON "Tag"("isSystem");

-- CompanyTag
CREATE UNIQUE INDEX "CompanyTag_companyId_tagId_key" ON "CompanyTag"("companyId", "tagId");
CREATE INDEX "CompanyTag_companyId_idx" ON "CompanyTag"("companyId");
CREATE INDEX "CompanyTag_tagId_idx" ON "CompanyTag"("tagId");

-- ContactTag
CREATE UNIQUE INDEX "ContactTag_contactId_tagId_key" ON "ContactTag"("contactId", "tagId");
CREATE INDEX "ContactTag_contactId_idx" ON "ContactTag"("contactId");
CREATE INDEX "ContactTag_tagId_idx" ON "ContactTag"("tagId");

-- DealTag
CREATE UNIQUE INDEX "DealTag_dealId_tagId_key" ON "DealTag"("dealId", "tagId");
CREATE INDEX "DealTag_dealId_idx" ON "DealTag"("dealId");
CREATE INDEX "DealTag_tagId_idx" ON "DealTag"("tagId");

-- Lifecycle History
CREATE INDEX "CompanyLifecycleHistory_companyId_createdAt_idx" ON "CompanyLifecycleHistory"("companyId", "createdAt" DESC);
CREATE INDEX "ContactLifecycleHistory_contactId_createdAt_idx" ON "ContactLifecycleHistory"("contactId", "createdAt" DESC);
CREATE INDEX "DealStageHistory_dealId_createdAt_idx" ON "DealStageHistory"("dealId", "createdAt" DESC);

-- Insights
CREATE INDEX "CompanyInsight_companyId_createdAt_idx" ON "CompanyInsight"("companyId", "createdAt" DESC);
CREATE INDEX "CompanyInsight_workspaceId_insightType_idx" ON "CompanyInsight"("workspaceId", "insightType");
CREATE INDEX "CompanyInsight_originStepId_idx" ON "CompanyInsight"("originStepId");

CREATE INDEX "ContactInsight_contactId_createdAt_idx" ON "ContactInsight"("contactId", "createdAt" DESC);
CREATE INDEX "ContactInsight_workspaceId_insightType_idx" ON "ContactInsight"("workspaceId", "insightType");
CREATE INDEX "ContactInsight_originStepId_idx" ON "ContactInsight"("originStepId");

-- Address
CREATE INDEX "Address_workspaceId_idx" ON "Address"("workspaceId");
CREATE INDEX "Address_contactId_deletedAt_orderIndex_idx" ON "Address"("contactId", "deletedAt", "orderIndex");
CREATE INDEX "Address_companyId_deletedAt_orderIndex_idx" ON "Address"("companyId", "deletedAt", "orderIndex");
CREATE INDEX "Address_phoneNumber_idx" ON "Address"("phoneNumber");

-- Idempotency
CREATE UNIQUE INDEX "idempotency_keys_key_key" ON "idempotency_keys"("key");
CREATE INDEX "idempotency_keys_workspaceId_idx" ON "idempotency_keys"("workspaceId");
CREATE INDEX "idempotency_keys_expiresAt_idx" ON "idempotency_keys"("expiresAt");

-- Audit Log
CREATE INDEX "audit_log_workspaceId_idx" ON "audit_log"("workspaceId");
CREATE INDEX "audit_log_userId_idx" ON "audit_log"("userId");
CREATE INDEX "audit_log_entityType_entityId_idx" ON "audit_log"("entityType", "entityId");
CREATE INDEX "audit_log_createdAt_idx" ON "audit_log"("createdAt" DESC);

-- =====================================================
-- 4️⃣ FOREIGN KEYS (CONSTRAINTS)
-- =====================================================

-- User
ALTER TABLE "User" ADD CONSTRAINT "User_adminRoleId_fkey" 
    FOREIGN KEY ("adminRoleId") REFERENCES "AdminRole"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- Workspace
ALTER TABLE "Workspace" ADD CONSTRAINT "Workspace_ownerId_fkey" 
    FOREIGN KEY ("ownerId") REFERENCES "User"("id") ON DELETE RESTRICT ON UPDATE CASCADE;
ALTER TABLE "Workspace" ADD CONSTRAINT "Workspace_organizationId_fkey" 
    FOREIGN KEY ("organizationId") REFERENCES "Organization"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- WorkspaceMember
ALTER TABLE "WorkspaceMember" ADD CONSTRAINT "WorkspaceMember_userId_fkey" 
    FOREIGN KEY ("userId") REFERENCES "User"("id") ON DELETE RESTRICT ON UPDATE CASCADE;
ALTER TABLE "WorkspaceMember" ADD CONSTRAINT "WorkspaceMember_workspaceId_fkey" 
    FOREIGN KEY ("workspaceId") REFERENCES "Workspace"("id") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "WorkspaceMember" ADD CONSTRAINT "WorkspaceMember_workspaceRoleId_fkey" 
    FOREIGN KEY ("workspaceRoleId") REFERENCES "WorkspaceRole"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- Contact
ALTER TABLE "Contact" ADD CONSTRAINT "Contact_workspaceId_fkey" 
    FOREIGN KEY ("workspaceId") REFERENCES "Workspace"("id") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "Contact" ADD CONSTRAINT "Contact_ownerId_fkey" 
    FOREIGN KEY ("ownerId") REFERENCES "User"("id") ON DELETE SET NULL ON UPDATE CASCADE;
ALTER TABLE "Contact" ADD CONSTRAINT "Contact_companyId_fkey" 
    FOREIGN KEY ("companyId") REFERENCES "Company"("id") ON DELETE SET NULL ON UPDATE CASCADE;
ALTER TABLE "Contact" ADD CONSTRAINT "Contact_assignedToId_fkey" 
    FOREIGN KEY ("assignedToId") REFERENCES "User"("id") ON DELETE SET NULL ON UPDATE CASCADE;
ALTER TABLE "Contact" ADD CONSTRAINT "Contact_createdById_fkey" 
    FOREIGN KEY ("createdById") REFERENCES "User"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- Company
ALTER TABLE "Company" ADD CONSTRAINT "Company_workspaceId_fkey" 
    FOREIGN KEY ("workspaceId") REFERENCES "Workspace"("id") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "Company" ADD CONSTRAINT "Company_assignedToId_fkey" 
    FOREIGN KEY ("assignedToId") REFERENCES "User"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- Pipeline
ALTER TABLE "Pipeline" ADD CONSTRAINT "Pipeline_workspaceId_fkey" 
    FOREIGN KEY ("workspaceId") REFERENCES "Workspace"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- PipelineStage
ALTER TABLE "PipelineStage" ADD CONSTRAINT "PipelineStage_pipelineId_fkey" 
    FOREIGN KEY ("pipelineId") REFERENCES "Pipeline"("id") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "PipelineStage" ADD CONSTRAINT "PipelineStage_workspaceId_fkey" 
    FOREIGN KEY ("workspaceId") REFERENCES "Workspace"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- Deal
ALTER TABLE "Deal" ADD CONSTRAINT "Deal_workspaceId_fkey" 
    FOREIGN KEY ("workspaceId") REFERENCES "Workspace"("id") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "Deal" ADD CONSTRAINT "Deal_pipelineId_fkey" 
    FOREIGN KEY ("pipelineId") REFERENCES "Pipeline"("id") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "Deal" ADD CONSTRAINT "Deal_stageId_fkey" 
    FOREIGN KEY ("stageId") REFERENCES "PipelineStage"("id") ON DELETE SET NULL ON UPDATE CASCADE;
ALTER TABLE "Deal" ADD CONSTRAINT "Deal_contactId_fkey" 
    FOREIGN KEY ("contactId") REFERENCES "Contact"("id") ON DELETE SET NULL ON UPDATE CASCADE;
ALTER TABLE "Deal" ADD CONSTRAINT "Deal_companyId_fkey" 
    FOREIGN KEY ("companyId") REFERENCES "Company"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- Task
ALTER TABLE "Task" ADD CONSTRAINT "Task_workspaceId_fkey" 
    FOREIGN KEY ("workspaceId") REFERENCES "Workspace"("id") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "Task" ADD CONSTRAINT "Task_companyId_fkey" 
    FOREIGN KEY ("companyId") REFERENCES "Company"("id") ON DELETE SET NULL ON UPDATE CASCADE;
ALTER TABLE "Task" ADD CONSTRAINT "Task_contactId_fkey" 
    FOREIGN KEY ("contactId") REFERENCES "Contact"("id") ON DELETE SET NULL ON UPDATE CASCADE;
ALTER TABLE "Task" ADD CONSTRAINT "Task_dealId_fkey" 
    FOREIGN KEY ("dealId") REFERENCES "Deal"("id") ON DELETE SET NULL ON UPDATE CASCADE;
ALTER TABLE "Task" ADD CONSTRAINT "Task_assignedToId_fkey" 
    FOREIGN KEY ("assignedToId") REFERENCES "User"("id") ON DELETE SET NULL ON UPDATE CASCADE;
ALTER TABLE "Task" ADD CONSTRAINT "Task_stageId_fkey" 
    FOREIGN KEY ("stageId") REFERENCES "PipelineStage"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- TaskComment
ALTER TABLE "TaskComment" ADD CONSTRAINT "TaskComment_taskId_fkey" 
    FOREIGN KEY ("taskId") REFERENCES "Task"("id") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "TaskComment" ADD CONSTRAINT "TaskComment_userId_fkey" 
    FOREIGN KEY ("userId") REFERENCES "User"("id") ON DELETE RESTRICT ON UPDATE CASCADE;
ALTER TABLE "TaskComment" ADD CONSTRAINT "TaskComment_workspaceId_fkey" 
    FOREIGN KEY ("workspaceId") REFERENCES "Workspace"("id") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "TaskComment" ADD CONSTRAINT "TaskComment_parentId_fkey" 
    FOREIGN KEY ("parentId") REFERENCES "TaskComment"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- Activity
ALTER TABLE "Activity" ADD CONSTRAINT "Activity_workspaceId_fkey" 
    FOREIGN KEY ("workspaceId") REFERENCES "Workspace"("id") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "Activity" ADD CONSTRAINT "Activity_userId_fkey" 
    FOREIGN KEY ("userId") REFERENCES "User"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- Note
ALTER TABLE "Note" ADD CONSTRAINT "Note_workspaceId_fkey" 
    FOREIGN KEY ("workspaceId") REFERENCES "Workspace"("id") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "Note" ADD CONSTRAINT "Note_userId_fkey" 
    FOREIGN KEY ("userId") REFERENCES "User"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- Message
ALTER TABLE "Message" ADD CONSTRAINT "Message_workspaceId_fkey" 
    FOREIGN KEY ("workspaceId") REFERENCES "Workspace"("id") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "Message" ADD CONSTRAINT "Message_contactId_fkey" 
    FOREIGN KEY ("contactId") REFERENCES "Contact"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- Email
ALTER TABLE "Email" ADD CONSTRAINT "Email_workspaceId_fkey" 
    FOREIGN KEY ("workspaceId") REFERENCES "Workspace"("id") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "Email" ADD CONSTRAINT "Email_contactId_fkey" 
    FOREIGN KEY ("contactId") REFERENCES "Contact"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- Call
ALTER TABLE "Call" ADD CONSTRAINT "Call_workspaceId_fkey" 
    FOREIGN KEY ("workspaceId") REFERENCES "Workspace"("id") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "Call" ADD CONSTRAINT "Call_contactId_fkey" 
    FOREIGN KEY ("contactId") REFERENCES "Contact"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- Meeting
ALTER TABLE "Meeting" ADD CONSTRAINT "Meeting_workspaceId_fkey" 
    FOREIGN KEY ("workspaceId") REFERENCES "Workspace"("id") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "Meeting" ADD CONSTRAINT "Meeting_userId_fkey" 
    FOREIGN KEY ("userId") REFERENCES "User"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- MeetingAttendee
ALTER TABLE "MeetingAttendee" ADD CONSTRAINT "MeetingAttendee_meetingId_fkey" 
    FOREIGN KEY ("meetingId") REFERENCES "Meeting"("id") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "MeetingAttendee" ADD CONSTRAINT "MeetingAttendee_contactId_fkey" 
    FOREIGN KEY ("contactId") REFERENCES "Contact"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- Tag
ALTER TABLE "Tag" ADD CONSTRAINT "Tag_workspaceId_fkey" 
    FOREIGN KEY ("workspaceId") REFERENCES "Workspace"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- CompanyTag, ContactTag, DealTag
ALTER TABLE "CompanyTag" ADD CONSTRAINT "CompanyTag_companyId_fkey" 
    FOREIGN KEY ("companyId") REFERENCES "Company"("id") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "CompanyTag" ADD CONSTRAINT "CompanyTag_tagId_fkey" 
    FOREIGN KEY ("tagId") REFERENCES "Tag"("id") ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE "ContactTag" ADD CONSTRAINT "ContactTag_contactId_fkey" 
    FOREIGN KEY ("contactId") REFERENCES "Contact"("id") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "ContactTag" ADD CONSTRAINT "ContactTag_tagId_fkey" 
    FOREIGN KEY ("tagId") REFERENCES "Tag"("id") ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE "DealTag" ADD CONSTRAINT "DealTag_dealId_fkey" 
    FOREIGN KEY ("dealId") REFERENCES "Deal"("id") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "DealTag" ADD CONSTRAINT "DealTag_tagId_fkey" 
    FOREIGN KEY ("tagId") REFERENCES "Tag"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- Lifecycle History
ALTER TABLE "CompanyLifecycleHistory" ADD CONSTRAINT "CompanyLifecycleHistory_workspaceId_fkey" 
    FOREIGN KEY ("workspaceId") REFERENCES "Workspace"("id") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "CompanyLifecycleHistory" ADD CONSTRAINT "CompanyLifecycleHistory_companyId_fkey" 
    FOREIGN KEY ("companyId") REFERENCES "Company"("id") ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE "ContactLifecycleHistory" ADD CONSTRAINT "ContactLifecycleHistory_workspaceId_fkey" 
    FOREIGN KEY ("workspaceId") REFERENCES "Workspace"("id") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "ContactLifecycleHistory" ADD CONSTRAINT "ContactLifecycleHistory_contactId_fkey" 
    FOREIGN KEY ("contactId") REFERENCES "Contact"("id") ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE "DealStageHistory" ADD CONSTRAINT "DealStageHistory_workspaceId_fkey" 
    FOREIGN KEY ("workspaceId") REFERENCES "Workspace"("id") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "DealStageHistory" ADD CONSTRAINT "DealStageHistory_dealId_fkey" 
    FOREIGN KEY ("dealId") REFERENCES "Deal"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- Insights
ALTER TABLE "CompanyInsight" ADD CONSTRAINT "CompanyInsight_workspaceId_fkey" 
    FOREIGN KEY ("workspaceId") REFERENCES "Workspace"("id") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "CompanyInsight" ADD CONSTRAINT "CompanyInsight_companyId_fkey" 
    FOREIGN KEY ("companyId") REFERENCES "Company"("id") ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE "ContactInsight" ADD CONSTRAINT "ContactInsight_workspaceId_fkey" 
    FOREIGN KEY ("workspaceId") REFERENCES "Workspace"("id") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "ContactInsight" ADD CONSTRAINT "ContactInsight_contactId_fkey" 
    FOREIGN KEY ("contactId") REFERENCES "Contact"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- Followup Plans
ALTER TABLE "followup_plans" ADD CONSTRAINT "followup_plans_workspaceId_fkey" 
    FOREIGN KEY ("workspaceId") REFERENCES "Workspace"("id") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "followup_plans" ADD CONSTRAINT "followup_plans_contactId_fkey" 
    FOREIGN KEY ("contactId") REFERENCES "Contact"("id") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "followup_plans" ADD CONSTRAINT "followup_plans_companyId_fkey" 
    FOREIGN KEY ("companyId") REFERENCES "Company"("id") ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE "followup_steps" ADD CONSTRAINT "followup_steps_planId_fkey" 
    FOREIGN KEY ("planId") REFERENCES "followup_plans"("id") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "followup_steps" ADD CONSTRAINT "followup_steps_workspaceId_fkey" 
    FOREIGN KEY ("workspaceId") REFERENCES "Workspace"("id") ON DELETE CASCADE ON UPDATE CASCADE;

-- Address
ALTER TABLE "Address" ADD CONSTRAINT "Address_workspaceId_fkey" 
    FOREIGN KEY ("workspaceId") REFERENCES "Workspace"("id") ON DELETE CASCADE ON UPDATE CASCADE;
ALTER TABLE "Address" ADD CONSTRAINT "Address_createdById_fkey" 
    FOREIGN KEY ("createdById") REFERENCES "User"("id") ON DELETE RESTRICT ON UPDATE CASCADE;
ALTER TABLE "Address" ADD CONSTRAINT "Address_contactId_fkey" 
    FOREIGN KEY ("contactId") REFERENCES "Contact"("id") ON DELETE SET NULL ON UPDATE CASCADE;
ALTER TABLE "Address" ADD CONSTRAINT "Address_companyId_fkey" 
    FOREIGN KEY ("companyId") REFERENCES "Company"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- =====================================================
-- 5️⃣ SEEDS/DADOS INICIAIS (Se houver)
-- =====================================================

-- Workspace Roles (Padrão)
INSERT INTO "WorkspaceRole" ("id", "name") VALUES 
    ('clworkspace_admin', 'work_admin'),
    ('clworkspace_manager', 'work_manager'),
    ('clworkspace_user', 'work_user')
ON CONFLICT ("name") DO NOTHING;

-- Admin Roles (Plataforma)
INSERT INTO "AdminRole" ("id", "name") VALUES 
    ('cladmin_super', 'super_admin'),
    ('cladmin_admin', 'admin'),
    ('cladmin_manager', 'manager')
ON CONFLICT ("name") DO NOTHING;

-- =====================================================
-- FIM DO SCHEMA SQL
-- =====================================================

-- NOTA: Este arquivo contém APENAS as tabelas principais do CRM.
-- Tabelas adicionais de AI, Portfolio, Templates estão no schema 
-- completo (schema-sql-output.sql)
--
-- Para aplicar este schema:
-- 1. Crie um banco PostgreSQL vazio
-- 2. Execute: psql -d seu_banco -f NEXTCRM_DATABASE_SCHEMAS_SQL.sql
-- 3. Configure a DATABASE_URL no .env
-- 4. Execute: npx prisma generate
--
-- Desenvolvido por NextCRM Team
-- GitHub: github.com/linkkotech/nextcrm-primeiro

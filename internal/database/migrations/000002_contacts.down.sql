-- Drop contacts table
DROP TRIGGER IF EXISTS update_contacts_updated_at ON contacts;
DROP FUNCTION IF EXISTS update_updated_at_column;
DROP INDEX IF EXISTS idx_contacts_search;
DROP INDEX IF EXISTS idx_contacts_created_at;
DROP INDEX IF EXISTS idx_contacts_email;
DROP INDEX IF EXISTS idx_contacts_company;
DROP INDEX IF EXISTS idx_contacts_owner;
DROP INDEX IF EXISTS idx_contacts_workspace;
DROP TABLE IF EXISTS contacts;

-- 000002_portfolio_schema.down.sql
-- Reverse dependency order: referencing tables before referenced tables.

DROP TABLE IF EXISTS experience_skills;
DROP TABLE IF EXISTS project_skills;
DROP TABLE IF EXISTS chat_messages;
DROP TABLE IF EXISTS chat_sessions;
DROP TABLE IF EXISTS contact_messages;
DROP TABLE IF EXISTS social_links;
DROP TABLE IF EXISTS social_platforms;
DROP TABLE IF EXISTS achievements;
DROP TABLE IF EXISTS certifications;
DROP TABLE IF EXISTS education;
DROP TABLE IF EXISTS experience;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS skills;
DROP TABLE IF EXISTS skill_categories;
DROP TABLE IF EXISTS profile;

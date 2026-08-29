-- 000002_portfolio_schema.up.sql
-- Portfolio content + application data schema (no seed data).
-- Spec: docs/database/ddl-design.md
-- Creation order: referenced tables before referencing tables.

-- ---------------------------------------------------------------------------
-- 1. profile (singleton)
-- ---------------------------------------------------------------------------
CREATE TABLE profile (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lock_key     BOOLEAN NOT NULL DEFAULT TRUE,
    full_name    TEXT NOT NULL,
    headline     TEXT NOT NULL,
    summary      TEXT NOT NULL,
    location     TEXT,
    email_public TEXT,
    avatar_url   TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT profile_lock_key_true CHECK (lock_key),
    CONSTRAINT profile_lock_key_unique UNIQUE (lock_key),
    CONSTRAINT profile_full_name_nonempty CHECK (full_name <> ''),
    CONSTRAINT profile_headline_nonempty CHECK (headline <> ''),
    CONSTRAINT profile_summary_nonempty CHECK (summary <> ''),
    CONSTRAINT profile_email_public_nonempty CHECK (email_public IS NULL OR email_public <> ''),
    CONSTRAINT profile_avatar_url_nonempty CHECK (avatar_url IS NULL OR avatar_url <> ''),
    CONSTRAINT profile_location_nonempty CHECK (location IS NULL OR location <> '')
);

-- ---------------------------------------------------------------------------
-- 2. skill_categories
-- ---------------------------------------------------------------------------
CREATE TABLE skill_categories (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug          TEXT NOT NULL,
    name          TEXT NOT NULL,
    display_order INTEGER NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT skill_categories_slug_unique UNIQUE (slug),
    CONSTRAINT skill_categories_name_unique UNIQUE (name),
    CONSTRAINT skill_categories_slug_nonempty CHECK (slug <> ''),
    CONSTRAINT skill_categories_name_nonempty CHECK (name <> ''),
    CONSTRAINT skill_categories_display_order_nonnegative CHECK (display_order >= 0)
);

-- ---------------------------------------------------------------------------
-- 3. skills
-- ---------------------------------------------------------------------------
CREATE TABLE skills (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id   UUID NOT NULL REFERENCES skill_categories (id) ON DELETE RESTRICT,
    slug          TEXT NOT NULL,
    name          TEXT NOT NULL,
    logo_url      TEXT,
    is_featured   BOOLEAN NOT NULL DEFAULT FALSE,
    is_published  BOOLEAN NOT NULL DEFAULT FALSE,
    display_order INTEGER NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT skills_slug_unique UNIQUE (slug),
    CONSTRAINT skills_name_unique UNIQUE (name),
    CONSTRAINT skills_slug_nonempty CHECK (slug <> ''),
    CONSTRAINT skills_name_nonempty CHECK (name <> ''),
    CONSTRAINT skills_logo_url_nonempty CHECK (logo_url IS NULL OR logo_url <> ''),
    CONSTRAINT skills_display_order_nonnegative CHECK (display_order >= 0)
);

CREATE INDEX idx_skills_category_id ON skills (category_id);

-- ---------------------------------------------------------------------------
-- 4. projects
-- ---------------------------------------------------------------------------
CREATE TABLE projects (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug          TEXT NOT NULL,
    title         TEXT NOT NULL,
    summary       TEXT NOT NULL,
    description   TEXT NOT NULL,
    role          TEXT NOT NULL,
    project_type  TEXT NOT NULL,
    repo_url      TEXT,
    live_url      TEXT,
    image_url     TEXT,
    started_on    DATE,
    ended_on      DATE,
    is_featured   BOOLEAN NOT NULL DEFAULT FALSE,
    is_published  BOOLEAN NOT NULL DEFAULT FALSE,
    display_order INTEGER NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT projects_slug_unique UNIQUE (slug),
    CONSTRAINT projects_slug_nonempty CHECK (slug <> ''),
    CONSTRAINT projects_title_nonempty CHECK (title <> ''),
    CONSTRAINT projects_summary_nonempty CHECK (summary <> ''),
    CONSTRAINT projects_description_nonempty CHECK (description <> ''),
    CONSTRAINT projects_role_valid CHECK (
        role IN ('solo', 'lead', 'contributor', 'maintainer')
    ),
    CONSTRAINT projects_project_type_valid CHECK (
        project_type IN ('solo', 'team', 'open_source', 'client', 'academic')
    ),
    CONSTRAINT projects_repo_url_nonempty CHECK (repo_url IS NULL OR repo_url <> ''),
    CONSTRAINT projects_live_url_nonempty CHECK (live_url IS NULL OR live_url <> ''),
    CONSTRAINT projects_image_url_nonempty CHECK (image_url IS NULL OR image_url <> ''),
    CONSTRAINT projects_dates_valid CHECK (
        ended_on IS NULL OR started_on IS NULL OR ended_on >= started_on
    ),
    CONSTRAINT projects_display_order_nonnegative CHECK (display_order >= 0)
);

-- ---------------------------------------------------------------------------
-- 5. experience
-- ---------------------------------------------------------------------------
CREATE TABLE experience (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company         TEXT NOT NULL,
    title           TEXT NOT NULL,
    location        TEXT,
    logo_url        TEXT,
    employment_type TEXT,
    start_date      DATE NOT NULL,
    end_date        DATE,
    description     TEXT NOT NULL,
    is_published    BOOLEAN NOT NULL DEFAULT FALSE,
    display_order   INTEGER NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT experience_company_nonempty CHECK (company <> ''),
    CONSTRAINT experience_title_nonempty CHECK (title <> ''),
    CONSTRAINT experience_description_nonempty CHECK (description <> ''),
    CONSTRAINT experience_location_nonempty CHECK (location IS NULL OR location <> ''),
    CONSTRAINT experience_logo_url_nonempty CHECK (logo_url IS NULL OR logo_url <> ''),
    CONSTRAINT experience_employment_type_valid CHECK (
        employment_type IS NULL OR employment_type IN (
            'full_time', 'part_time', 'contract', 'internship', 'freelance', 'other'
        )
    ),
    CONSTRAINT experience_dates_valid CHECK (
        end_date IS NULL OR end_date >= start_date
    ),
    CONSTRAINT experience_display_order_nonnegative CHECK (display_order >= 0)
);

-- ---------------------------------------------------------------------------
-- 6. education
-- ---------------------------------------------------------------------------
CREATE TABLE education (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution    TEXT NOT NULL,
    degree         TEXT NOT NULL,
    field_of_study TEXT,
    location       TEXT,
    start_date     DATE NOT NULL,
    end_date       DATE,
    description    TEXT,
    is_published   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT education_institution_nonempty CHECK (institution <> ''),
    CONSTRAINT education_degree_nonempty CHECK (degree <> ''),
    CONSTRAINT education_field_of_study_nonempty CHECK (
        field_of_study IS NULL OR field_of_study <> ''
    ),
    CONSTRAINT education_location_nonempty CHECK (location IS NULL OR location <> ''),
    CONSTRAINT education_description_nonempty CHECK (
        description IS NULL OR description <> ''
    ),
    CONSTRAINT education_dates_valid CHECK (
        end_date IS NULL OR end_date >= start_date
    )
);

-- ---------------------------------------------------------------------------
-- 7. certifications
-- ---------------------------------------------------------------------------
CREATE TABLE certifications (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name           TEXT NOT NULL,
    issuer         TEXT NOT NULL,
    credential_id  TEXT,
    credential_url TEXT,
    issued_on      DATE NOT NULL,
    expires_on     DATE,
    is_published   BOOLEAN NOT NULL DEFAULT FALSE,
    display_order  INTEGER NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT certifications_name_nonempty CHECK (name <> ''),
    CONSTRAINT certifications_issuer_nonempty CHECK (issuer <> ''),
    CONSTRAINT certifications_credential_id_nonempty CHECK (
        credential_id IS NULL OR credential_id <> ''
    ),
    CONSTRAINT certifications_credential_url_nonempty CHECK (
        credential_url IS NULL OR credential_url <> ''
    ),
    CONSTRAINT certifications_dates_valid CHECK (
        expires_on IS NULL OR expires_on >= issued_on
    ),
    CONSTRAINT certifications_display_order_nonnegative CHECK (display_order >= 0)
);

-- ---------------------------------------------------------------------------
-- 8. achievements
-- ---------------------------------------------------------------------------
CREATE TABLE achievements (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title         TEXT NOT NULL,
    description   TEXT,
    achieved_on   DATE,
    url           TEXT,
    is_published  BOOLEAN NOT NULL DEFAULT FALSE,
    display_order INTEGER NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT achievements_title_nonempty CHECK (title <> ''),
    CONSTRAINT achievements_description_nonempty CHECK (
        description IS NULL OR description <> ''
    ),
    CONSTRAINT achievements_url_nonempty CHECK (url IS NULL OR url <> ''),
    CONSTRAINT achievements_display_order_nonnegative CHECK (display_order >= 0)
);

-- ---------------------------------------------------------------------------
-- 9. social_platforms
-- ---------------------------------------------------------------------------
CREATE TABLE social_platforms (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug          TEXT NOT NULL,
    name          TEXT NOT NULL,
    logo_url      TEXT,
    display_order INTEGER NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT social_platforms_slug_unique UNIQUE (slug),
    CONSTRAINT social_platforms_name_unique UNIQUE (name),
    CONSTRAINT social_platforms_slug_nonempty CHECK (slug <> ''),
    CONSTRAINT social_platforms_name_nonempty CHECK (name <> ''),
    CONSTRAINT social_platforms_logo_url_nonempty CHECK (
        logo_url IS NULL OR logo_url <> ''
    ),
    CONSTRAINT social_platforms_slug_not_email CHECK (slug <> 'email'),
    CONSTRAINT social_platforms_display_order_nonnegative CHECK (display_order >= 0)
);

-- ---------------------------------------------------------------------------
-- 10. social_links
-- ---------------------------------------------------------------------------
CREATE TABLE social_links (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    platform_id   UUID NOT NULL REFERENCES social_platforms (id) ON DELETE RESTRICT,
    label         TEXT,
    url           TEXT NOT NULL,
    logo_url      TEXT,
    is_published  BOOLEAN NOT NULL DEFAULT FALSE,
    display_order INTEGER NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT social_links_platform_id_unique UNIQUE (platform_id),
    CONSTRAINT social_links_url_nonempty CHECK (url <> ''),
    CONSTRAINT social_links_label_nonempty CHECK (label IS NULL OR label <> ''),
    CONSTRAINT social_links_logo_url_nonempty CHECK (logo_url IS NULL OR logo_url <> ''),
    CONSTRAINT social_links_display_order_nonnegative CHECK (display_order >= 0)
);

-- ---------------------------------------------------------------------------
-- 11. contact_messages
-- ---------------------------------------------------------------------------
CREATE TABLE contact_messages (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    email      TEXT NOT NULL,
    subject    TEXT,
    body       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT contact_messages_name_nonempty CHECK (name <> ''),
    CONSTRAINT contact_messages_email_nonempty CHECK (email <> ''),
    CONSTRAINT contact_messages_body_nonempty CHECK (body <> ''),
    CONSTRAINT contact_messages_subject_nonempty CHECK (
        subject IS NULL OR subject <> ''
    )
);

CREATE INDEX idx_contact_messages_created_at ON contact_messages (created_at DESC);

-- ---------------------------------------------------------------------------
-- 12. chat_sessions
-- ---------------------------------------------------------------------------
CREATE TABLE chat_sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    visitor_key     TEXT NOT NULL,
    started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_message_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at        TIMESTAMPTZ,
    metadata        JSONB,

    CONSTRAINT chat_sessions_visitor_key_nonempty CHECK (visitor_key <> ''),
    CONSTRAINT chat_sessions_last_message_at_valid CHECK (last_message_at >= started_at),
    CONSTRAINT chat_sessions_ended_at_valid CHECK (
        ended_at IS NULL OR ended_at >= started_at
    )
);

CREATE INDEX idx_chat_sessions_visitor_key ON chat_sessions (visitor_key);
CREATE INDEX idx_chat_sessions_last_message_at ON chat_sessions (last_message_at DESC);

-- ---------------------------------------------------------------------------
-- 13. chat_messages
-- ---------------------------------------------------------------------------
CREATE TABLE chat_messages (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id  UUID NOT NULL REFERENCES chat_sessions (id) ON DELETE CASCADE,
    role        TEXT NOT NULL,
    content     TEXT NOT NULL,
    token_count INTEGER,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chat_messages_role_valid CHECK (
        role IN ('user', 'assistant', 'system')
    ),
    CONSTRAINT chat_messages_content_nonempty CHECK (content <> ''),
    CONSTRAINT chat_messages_token_count_nonnegative CHECK (
        token_count IS NULL OR token_count >= 0
    )
);

CREATE INDEX idx_chat_messages_session_created
    ON chat_messages (session_id, created_at ASC);

-- ---------------------------------------------------------------------------
-- 14. project_skills
-- ---------------------------------------------------------------------------
CREATE TABLE project_skills (
    project_id    UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    skill_id      UUID NOT NULL REFERENCES skills (id) ON DELETE CASCADE,
    display_order INTEGER NOT NULL DEFAULT 0,

    CONSTRAINT project_skills_pkey PRIMARY KEY (project_id, skill_id),
    CONSTRAINT project_skills_display_order_nonnegative CHECK (display_order >= 0)
);

CREATE INDEX idx_project_skills_skill_id ON project_skills (skill_id);

-- ---------------------------------------------------------------------------
-- 15. experience_skills
-- ---------------------------------------------------------------------------
CREATE TABLE experience_skills (
    experience_id UUID NOT NULL REFERENCES experience (id) ON DELETE CASCADE,
    skill_id      UUID NOT NULL REFERENCES skills (id) ON DELETE CASCADE,
    display_order INTEGER NOT NULL DEFAULT 0,

    CONSTRAINT experience_skills_pkey PRIMARY KEY (experience_id, skill_id),
    CONSTRAINT experience_skills_display_order_nonnegative CHECK (display_order >= 0)
);

CREATE INDEX idx_experience_skills_skill_id ON experience_skills (skill_id);

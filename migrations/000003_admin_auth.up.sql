-- 000003_admin_auth.up.sql
-- Singleton admin credential + single active session.
-- Spec: docs/admin/architecture.md
-- No seed data — password is set via cmd/admin (later).

CREATE TABLE admin (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lock_key                 BOOLEAN NOT NULL DEFAULT TRUE,
    password_hash            TEXT NOT NULL,
    access_token_hash        TEXT,
    refresh_token_hash       TEXT,
    access_token_expires_at  TIMESTAMPTZ,
    refresh_token_expires_at TIMESTAMPTZ,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT admin_lock_key_true CHECK (lock_key),
    CONSTRAINT admin_lock_key_unique UNIQUE (lock_key),
    CONSTRAINT admin_password_hash_nonempty CHECK (password_hash <> ''),
    CONSTRAINT admin_access_token_hash_nonempty CHECK (
        access_token_hash IS NULL OR access_token_hash <> ''
    ),
    CONSTRAINT admin_refresh_token_hash_nonempty CHECK (
        refresh_token_hash IS NULL OR refresh_token_hash <> ''
    ),
    CONSTRAINT admin_access_session_pair CHECK (
        (access_token_hash IS NULL) = (access_token_expires_at IS NULL)
    ),
    CONSTRAINT admin_refresh_session_pair CHECK (
        (refresh_token_hash IS NULL) = (refresh_token_expires_at IS NULL)
    )
);

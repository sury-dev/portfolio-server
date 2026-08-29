# Admin Architecture

**Scope:** Single-operator authentication and authorized CRUD over portfolio content.  
**Status:** Design (revised)  
**Related:** [`../database/domain-model.md`](../database/domain-model.md), [`../database/ddl-design.md`](../database/ddl-design.md), [`workflows.md`](./workflows.md)

This document answers:

> How does the sole admin authenticate, where do credentials live, and how does a new login invalidate previous sessions?

---

## 1. Design premise

There is **exactly one admin**: you.

| Multi-tenant admin CMS | This portfolio |
|------------------------|----------------|
| Many admin users | **One** `admin` row (singleton) |
| Roles / permissions | **No** |
| Password hash in config | Password hash in **`admin` table** |
| Many concurrent sessions | **One active session** — login overwrites tokens |
| Admin CLI | Sets / rotates password hash in DB |

Visitors stay read-only (plus contact / chat). Only the admin mutates portfolio content.

```text
Admin CLI (offline / ops)
    │
    └── hashes password → UPSERT admin.password_hash


Admin API
    │
    ├── POST /admin/auth/login    → verify DB hash, overwrite session tokens
    ├── POST /admin/auth/refresh  → validate current refresh, rotate tokens
    ├── POST /admin/auth/logout   → clear session tokens on the admin row
    └── CRUD /admin/...           → Bearer must match current session
```

---

## 2. Separation of concerns

```text
┌──────────────────────────────────────────────────────────┐
│  Admin CLI (cmd/admin)                                   │
│  Input: password (prompt / flag)                         │
│  Output: password_hash written to PostgreSQL `admin`     │
└────────────────────────────┬─────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────┐
│  PostgreSQL — table `admin` (exactly one row)            │
│  password_hash                                           │
│  current access + refresh token material (overwritten)   │
└────────────────────────────┬─────────────────────────────┘
                             │
┌────────────────────────────┴─────────────────────────────┐
│  Config (signing / TTL only — NOT the password)          │
│  JWT_ACCESS_SECRET / JWT_REFRESH_SECRET (if JWT used)    │
│  ACCESS_TOKEN_TTL / REFRESH_TOKEN_TTL                    │
└────────────────────────────┬─────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────┐
│  Auth service + Fiber /admin routes                      │
│  login / refresh / logout / CRUD middleware              │
└──────────────────────────────────────────────────────────┘
```

**Password never lives in config.**  
Config may still hold **token signing secrets and TTLs** — those are infrastructure secrets, not the admin credential.

---

## 3. The `admin` table (singleton)

Same singleton idea as `profile`: enforce **at most one row** in the database.

### 3.1 Conceptual columns

```text
Table: admin

Column                    Type          Null?    Notes
-------------------------------------------------------------------
id                        UUID          NO       PK, default gen_random_uuid()
lock_key                  BOOLEAN       NO       DEFAULT TRUE; UNIQUE; CHECK (lock_key)
password_hash             TEXT          NO       argon2id / bcrypt hash
access_token_hash         TEXT          YES      hash of current access token (NULL = logged out)
refresh_token_hash        TEXT          YES      hash of current refresh token
access_token_expires_at   TIMESTAMPTZ   YES
refresh_token_expires_at  TIMESTAMPTZ   YES
created_at                TIMESTAMPTZ   NO       DEFAULT NOW()
updated_at                TIMESTAMPTZ   NO       DEFAULT NOW()
```

### 3.2 Singleton enforcement

```text
lock_key BOOLEAN NOT NULL DEFAULT TRUE
CHECK (lock_key)
UNIQUE (lock_key)
```

Identical pattern to `profile.lock_key`.

### 3.3 What is stored vs returned

| Stored in DB | Returned to client |
|--------------|--------------------|
| `password_hash` | never |
| `access_token_hash` | never (raw access token once at login/refresh) |
| `refresh_token_hash` | never (raw refresh token once at login/refresh) |

Always store **hashes** of tokens (e.g. SHA-256 of the raw token), not the raw tokens themselves.

### 3.4 Migration placement

`admin` is application auth data, not portfolio content. Add in a dedicated migration after `000002`, e.g.:

```text
000003_admin_auth
```

Do not fold it into portfolio content tables, and do not add `created_by` on content rows.

---

## 4. Admin CLI

### 4.1 Responsibility

A small Go command (e.g. `cmd/admin`) whose **only** credential job is:

```text
read password → hash → upsert into admin.password_hash
```

It is **not** the HTTP API. It does not mint tokens. It does not serve CRUD.

### 4.2 Suggested commands

```text
admin set-password          # prompt (or --password for automation; prefer prompt)
admin status                # row exists? password set? session active? (no secrets printed)
```

### 4.3 Behavior of `set-password`

```text
1. Connect using existing DATABASE config
2. Hash password with argon2id (preferred) or bcrypt
3. UPSERT the singleton admin row:
     - if no row → INSERT (password_hash, lock_key=TRUE)
     - if row exists → UPDATE password_hash, updated_at
4. Clear session columns (access/refresh hashes + expiries)
   → forces re-login after password change
5. Exit non-zero on failure; never print the hash unless an explicit debug flag is used
```

### 4.4 What the CLI must not do

- Write password or hash into `config.conf`
- Accept password via world-readable env in production docs as the primary path
- Create multiple admin rows
- Bypass singleton constraints

---

## 5. Token model — single active session

### 5.1 Rule

**Any successful login overwrites the session token fields on the `admin` row.**

```text
Login A  →  tokens_A written to admin
Login B  →  tokens_B overwrite tokens_A
                 │
                 └── tokens_A are dead (hash no longer matches)
```

There is no multi-session table. The singleton row **is** the session.

### 5.2 Access token

| Property | Choice |
|----------|--------|
| Format | JWT (signed with config secret) **or** opaque random string |
| Lifetime | Short (minutes) |
| Client header | `Authorization: Bearer <access_token>` |
| Server validation | Signature/expiry (if JWT) **and** `hash(token) == admin.access_token_hash` **and** not expired |

DB check is mandatory so an overwritten session cannot keep using an old access token until TTL.

### 5.3 Refresh token

| Property | Choice |
|----------|--------|
| Format | Opaque random string (recommended) |
| Lifetime | Longer (days) |
| Purpose | Mint a new access token without password |
| Server validation | `hash(token) == admin.refresh_token_hash` and not expired |

Refresh also **overwrites** access (and typically rotates refresh) on the same row.

### 5.4 Logout

```text
SET access_token_hash = NULL,
    refresh_token_hash = NULL,
    access_token_expires_at = NULL,
    refresh_token_expires_at = NULL
```

No separate revoke table.

---

## 6. Authentication flow (summary)

```text
CLI set-password
  → admin.password_hash = H(password)
  → session columns cleared


Login
  password
       │
       ▼
  load singleton admin row
  verify(password, password_hash)
       │
       ├── fail → 401
       └── ok  → mint access + refresh
                OVERWRITE token hashes + expiries on admin row
                return raw tokens to client


Authenticated request
  Bearer access
       │
       ▼
  verify token + hash match admin.access_token_hash + not expired
       │
       ├── fail → 401
       └── ok   → CRUD handler


Refresh
  refresh_token
       │
       ▼
  hash match admin.refresh_token_hash + not expired
       │
       ├── fail → 401
       └── ok  → mint new tokens, OVERWRITE row, return tokens


Logout
  clear token columns on admin row
```

Details: [`workflows.md`](./workflows.md).

---

## 7. Authorization model

Binary:

```text
Bearer matches current admin session ⇒ admin
otherwise                            ⇒ not admin
```

No roles. No per-resource ACLs in v1.

Public routes do not use this middleware.

---

## 8. Config that remains (non-password)

```text
[ADMIN]
JWT_ACCESS_SECRET = ...
JWT_REFRESH_SECRET = ...      # if refresh is also JWT; omit if opaque-only
ACCESS_TOKEN_TTL_SEC = 900
REFRESH_TOKEN_TTL_SEC = 604800
```

| In config | In database (`admin`) |
|-----------|------------------------|
| Signing secrets | `password_hash` |
| Token TTLs | Current session token hashes + expiries |

If both access and refresh are opaque DB-backed tokens, signing secrets can be omitted — still keep TTLs in config.

---

## 9. Admin CRUD surface

Unchanged intent: mutate portfolio content from `000002` behind auth.

```text
/admin/auth/login
/admin/auth/refresh
/admin/auth/logout

/admin/profile
/admin/skill-categories
/admin/skills
/admin/projects
/admin/projects/:id/skills
/admin/experience
/admin/experience/:id/skills
/admin/education
/admin/certifications
/admin/achievements
/admin/social-platforms
/admin/social-links
```

| Actor | Reads | Writes |
|-------|-------|--------|
| Public | `is_published = TRUE` | contact / chat only |
| Admin (current session) | all rows | full CRUD |

---

## 10. Security properties

| Control | Requirement |
|---------|-------------|
| Transport | HTTPS outside local dev |
| Password | Hash in DB only; set via CLI; strong password |
| Singleton | `lock_key` prevents a second admin row |
| Single session | Login overwrite invalidates previous tokens |
| Token at rest | Store hashes only |
| Brute force | Rate-limit `/admin/auth/login` |
| Logging | Never log password, raw tokens, or hashes |
| Password change | CLI clears session → must login again |

---

## 11. What we deliberately do **not** build

- Password hash in `config.conf`
- `admin_refresh_tokens` multi-row session table
- Multiple concurrent admin sessions
- `users` / roles / permissions
- OAuth / magic links / email reset
- Unauthenticated portfolio writes

---

## 12. Implementation sequence

```text
1. Migration 000003 — admin singleton table
2. cmd/admin — set-password (+ status)
3. Auth service — verify hash, mint/overwrite tokens
4. /admin/auth/* + Bearer middleware (DB session check)
5. First CRUD resource under /admin
6. Remaining CRUD resources
```

---

## 13. Schema timeline

```text
000001  infrastructure test
000002  portfolio + app data
000003  admin (singleton auth)   ← next auth migration
```
```

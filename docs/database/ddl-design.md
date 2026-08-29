# PostgreSQL DDL Design

**Task:** `003A` — PostgreSQL DDL Design  
**Branch:** `task/003_Schema_and_Models`  
**Status:** Approved (with corrections applied) — ready for `000002`  
**Depends on:** [`domain-model.md`](./domain-model.md)

This document answers:

> Exactly how will PostgreSQL represent the approved domain model?

It is the database contract. Migration `000002_portfolio_schema` must implement **this** design after review — not invent new columns or constraints.

---

## 1. Global conventions

### 1.1 Identifiers

| Rule | Choice |
|------|--------|
| Primary keys | `UUID NOT NULL DEFAULT gen_random_uuid()` |
| Extension | `pgcrypto` already enabled by `000001` (provides `gen_random_uuid()`) |
| Public deep-link slug | `projects.slug` only |
| Internal slugs | `skill_categories.slug`, `skills.slug`, `social_platforms.slug` — unique, not public URL contracts |
| Join identity | Composite PKs — no surrogate UUID on junction tables |

### 1.2 Text

Prefer `TEXT` over `VARCHAR(n)` unless a real business maximum exists.

Integrity rules for non-empty required strings:

```text
CHECK (column <> '')
```

UX length limits belong in the application layer.

### 1.3 Dates vs timestamps

| Kind | Type | Examples |
|------|------|----------|
| Calendar career / project dates | `DATE` | `started_on`, `start_date`, `issued_on` |
| Application event times | `TIMESTAMPTZ` | `created_at`, `started_at`, `last_message_at` |

### 1.4 Timestamps

Where mutable portfolio content exists:

```text
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
```

**PostgreSQL does not auto-bump `updated_at`.** That is an application/repository (or later trigger) concern.

Append-oriented application tables omit `updated_at` unless a real update concept exists:

| Table | `created_at` | `updated_at` |
|-------|--------------|--------------|
| Portfolio content tables | yes | yes |
| `contact_messages` | yes | **no** |
| `chat_sessions` | via `started_at` (+ `last_message_at`) | **no** `updated_at` |
| `chat_messages` | yes | **no** |

### 1.5 Visibility & featured

```text
is_published BOOLEAN NOT NULL DEFAULT FALSE
is_featured  BOOLEAN NOT NULL DEFAULT FALSE   -- only projects, skills
```

### 1.5.1 Display order

```text
display_order INTEGER NOT NULL DEFAULT 0
CHECK (display_order >= 0)
```

Negative ordering has no meaning for this portfolio. Apply on every table that uses `display_order`.

### 1.6 Controlled vocabularies

Prefer `TEXT + CHECK` over PostgreSQL `ENUM` for v1 (easier to evolve).

Lookup tables are used when the vocabulary needs **metadata** (name, logo, display order):

| Vocabulary | Mechanism |
|------------|-----------|
| Skill categories | `skill_categories` table |
| Social platforms | `social_platforms` table *(decided for v1)* |
| Project role / type | `TEXT + CHECK` |
| Employment type | `TEXT + CHECK` |
| Chat message role | `TEXT + CHECK` |

### 1.7 JSONB

Only on `chat_sessions.metadata` — semi-structured session hints. No JSONB on portfolio content.

### 1.8 Index philosophy

1. Unique constraints already create unique indexes — **do not duplicate** them with a second index on the same column.
2. Index FK columns that are looked up from the “many” side when the parent PK does not already lead the access path.
3. For composite PKs `(a, b)`, the leading column `a` is indexed; add a separate index on `b` when reverse lookups matter.
4. Defer speculative `(is_published, display_order)` composites until query volume justifies them.

### 1.9 Foreign keys: delete only, no update cascade

Primary keys are UUIDs and are treated as **immutable**. Do **not** specify `ON UPDATE CASCADE` (or any `ON UPDATE` action). Omit `ON UPDATE` and use PostgreSQL’s default (`NO ACTION`).

Only define meaningful **`ON DELETE`** behavior.

### 1.10 Deletion philosophy

| Pattern | ON DELETE | Why |
|---------|-----------|-----|
| Parent → private dependents (session → messages) | `CASCADE` | Dependents cannot exist alone |
| Parent → join rows (project → project_skills) | `CASCADE` | Join is ownership of a relationship |
| Shared catalog → join rows (skill → project_skills) | `CASCADE` | Removes **links only**, never projects/experience |
| Shared catalog → children that own meaning (category → skills, platform → links) | `RESTRICT` | Prevent silent destruction of domain content |

**Never** cascade from a shared skill/platform into destroying unrelated projects or experience rows.

---

## 2. Dependency graph & creation order

```text
                    skill_categories
                          │
                          ▼
                        skills
                       ▲     ▲
                       │     │
             ┌─────────┘     └─────────┐
             │                         │
         projects                  experience
             │                         │
             ▼                         ▼
      project_skills          experience_skills


                    social_platforms
                          │
                          ▼
                     social_links


Independent (no inbound FKs from others in v1):
  profile
  education
  certifications
  achievements
  contact_messages
  chat_sessions

Dependent:
  chat_messages → chat_sessions
```

### Up migration order

1. `profile`
2. `skill_categories`
3. `skills`
4. `projects`
5. `experience`
6. `education`
7. `certifications`
8. `achievements`
9. `social_platforms`
10. `social_links`
11. `contact_messages`
12. `chat_sessions`
13. `chat_messages`
14. `project_skills`
15. `experience_skills`

### Down migration order (reverse FKs)

1. `experience_skills`
2. `project_skills`
3. `chat_messages`
4. `chat_sessions`
5. `contact_messages`
6. `social_links`
7. `social_platforms`
8. `achievements`
9. `certifications`
10. `education`
11. `experience`
12. `projects`
13. `skills`
14. `skill_categories`
15. `profile`

---

## 3. Tables

---

### 3.1 `profile`

Singleton portfolio owner identity. Enforced **in the database**, not only in Go.

**Singleton mechanism chosen:** constant boolean lock column.

```text
lock_key BOOLEAN NOT NULL DEFAULT TRUE
CHECK (lock_key)           -- only TRUE allowed
UNIQUE (lock_key)          -- only one row can hold TRUE
```

Alternatives considered:

| Approach | Verdict |
|----------|---------|
| Unique index on `(TRUE)` expression | Works; less self-documenting as a column |
| PK fixed to `1` / discard UUID | Works; drops UUID consistency with other entities |
| Insert trigger counting rows | Heavier; avoid for v1 |
| App-only singleton | Rejected — invariant must live in PostgreSQL |

`id` remains a UUID for consistency and possible future references. Nothing FKs to `profile` in v1.

```text
Table: profile

Column          Type          Null?       Default
--------------------------------------------------------
id              UUID          NO          gen_random_uuid()
lock_key        BOOLEAN       NO          TRUE
full_name       TEXT          NO          -
headline        TEXT          NO          -
summary         TEXT          NO          -
location        TEXT          YES         -
email_public    TEXT          YES         -
avatar_url      TEXT          YES         -
created_at      TIMESTAMPTZ   NO          NOW()
updated_at      TIMESTAMPTZ   NO          NOW()
```

**Primary Key:** `id`

**Unique Constraints:**

- `UNIQUE (lock_key)` — singleton enforcement

**Foreign Keys:** none

**Check Constraints:**

- `lock_key` must be TRUE: `CHECK (lock_key)`
- `full_name <> ''`
- `headline <> ''`
- `summary <> ''`
- `email_public IS NULL OR email_public <> ''`
- `avatar_url IS NULL OR avatar_url <> ''`
- `location IS NULL OR location <> ''`

**Indexes:**

- PK on `id`
- Unique on `lock_key` (from UNIQUE constraint)
- No extra indexes

**Delete Behavior:**

- Hard delete is allowed by PostgreSQL but **should be rare** (would empty the site identity).
- No FK dependents in v1.
- Optional later hardening: revoke DELETE / add trigger — not required for first migration.

**Notes:**

- Email lives here (`email_public`), never as a social platform.

---

### 3.2 `skill_categories`

```text
Table: skill_categories

Column          Type          Null?       Default
--------------------------------------------------------
id              UUID          NO          gen_random_uuid()
slug            TEXT          NO          -
name            TEXT          NO          -
display_order   INTEGER       NO          0
created_at      TIMESTAMPTZ   NO          NOW()
updated_at      TIMESTAMPTZ   NO          NOW()
```

**Primary Key:** `id`

**Unique Constraints:**

- `UNIQUE (slug)`
- `UNIQUE (name)` — duplicate display labels (“Languages” twice) have no meaning

**Foreign Keys:** none

**Check Constraints:**

- `slug <> ''`
- `name <> ''`
- `display_order >= 0`

**Indexes:**

- PK; unique indexes from UNIQUE constraints
- No extra indexes

**Delete Behavior:**

- Referenced by `skills.category_id` → **`ON DELETE RESTRICT`**
- Cannot drop “Languages” while skills still point at it

**Intended list order:** `ORDER BY display_order ASC, name ASC`

---

### 3.3 `skills`

```text
Table: skills

Column          Type          Null?       Default
--------------------------------------------------------
id              UUID          NO          gen_random_uuid()
category_id     UUID          NO          -
slug            TEXT          NO          -
name            TEXT          NO          -
logo_url        TEXT          YES         -
is_featured     BOOLEAN       NO          FALSE
is_published    BOOLEAN       NO          FALSE
display_order   INTEGER       NO          0
created_at      TIMESTAMPTZ   NO          NOW()
updated_at      TIMESTAMPTZ   NO          NOW()
```

**Primary Key:** `id`

**Unique Constraints:**

- `UNIQUE (slug)` — internal stable key
- `UNIQUE (name)` — one canonical “Go” / “React” label

**Foreign Keys:**

| Column | References | ON DELETE |
|--------|------------|-----------|
| `category_id` | `skill_categories(id)` | **RESTRICT** |

**Check Constraints:**

- `slug <> ''`
- `name <> ''`
- `logo_url IS NULL OR logo_url <> ''`
- `display_order >= 0`

**Indexes:**

- PK; unique on `slug`, `name`
- **`INDEX (category_id)`** — list skills by category (`WHERE category_id = ?`)

**Delete Behavior:**

- Deleting a skill **CASCADE**s to `project_skills` and `experience_skills` (join rows only)
- Does **not** touch `projects` or `experience`

**Not present:** `proficiency`

**Slug contract:** internal only — not a public `/skills/...` deep link in v1

**Intended list order:** `ORDER BY display_order ASC, name ASC` (typically within a category)

---

### 3.4 `projects`

```text
Table: projects

Column          Type          Null?       Default
--------------------------------------------------------
id              UUID          NO          gen_random_uuid()
slug            TEXT          NO          -
title           TEXT          NO          -
summary         TEXT          NO          -
description     TEXT          NO          -
role            TEXT          NO          -
project_type    TEXT          NO          -
repo_url        TEXT          YES         -
live_url        TEXT          YES         -
image_url       TEXT          YES         -
started_on      DATE          YES         -
ended_on        DATE          YES         -
is_featured     BOOLEAN       NO          FALSE
is_published    BOOLEAN       NO          FALSE
display_order   INTEGER       NO          0
created_at      TIMESTAMPTZ   NO          NOW()
updated_at      TIMESTAMPTZ   NO          NOW()
```

**Primary Key:** `id`

**Unique Constraints:**

- `UNIQUE (slug)` — **public** deep-link contract

**Foreign Keys:** none (skills via join)

**Check Constraints:**

- `slug <> ''`
- `title <> ''`
- `summary <> ''`
- `description <> ''`
- `role IN ('solo', 'lead', 'contributor', 'maintainer')`
- `project_type IN ('solo', 'team', 'open_source', 'client', 'academic')`
- `repo_url IS NULL OR repo_url <> ''`
- `live_url IS NULL OR live_url <> ''`
- `image_url IS NULL OR image_url <> ''`
- Date integrity (ongoing allowed):

```text
ended_on IS NULL OR started_on IS NULL OR ended_on >= started_on
```

- `display_order >= 0`

**Indexes:**

- PK; unique on `slug`
- **No second index on `slug`**
- Deferred: `(is_published, display_order)` — personal portfolio size; add later if needed

**Delete Behavior:**

- Deleting a project **CASCADE**s to `project_skills`

**Slug notes:**

- Empty slug rejected
- URL shape validation (lowercase/kebab) can stay in the app for v1; DB enforces non-empty + unique

**Intended list order:** `ORDER BY display_order ASC, created_at DESC`

**Description:** rich markdown stored as `TEXT`

---

### 3.5 `experience`

```text
Table: experience

Column          Type          Null?       Default
--------------------------------------------------------
id              UUID          NO          gen_random_uuid()
company         TEXT          NO          -
title           TEXT          NO          -
location        TEXT          YES         -
logo_url        TEXT          YES         -
employment_type TEXT          YES         -
start_date      DATE          NO          -
end_date        DATE          YES         -
description     TEXT          NO          -
is_published    BOOLEAN       NO          FALSE
display_order   INTEGER       NO          0
created_at      TIMESTAMPTZ   NO          NOW()
updated_at      TIMESTAMPTZ   NO          NOW()
```

**Primary Key:** `id`

**Unique Constraints:** none

**Foreign Keys:** none (skills via join)

**Check Constraints:**

- `company <> ''`
- `title <> ''`
- `description <> ''`
- `location IS NULL OR location <> ''`
- `logo_url IS NULL OR logo_url <> ''`
- `employment_type IS NULL OR employment_type IN ('full_time', 'part_time', 'contract', 'internship', 'freelance', 'other')`
- Current role allowed:

```text
end_date IS NULL OR end_date >= start_date
```

- `display_order >= 0`

**Indexes:**

- PK only for v1
- Deferred: `(is_published, display_order)`

**Delete Behavior:**

- Deleting experience **CASCADE**s to `experience_skills`

**Not present:** `is_featured`

**Intended list order:** `ORDER BY display_order ASC, start_date DESC`

**Description:** rich markdown `TEXT`

---

### 3.6 `education`

```text
Table: education

Column          Type          Null?       Default
--------------------------------------------------------
id              UUID          NO          gen_random_uuid()
institution     TEXT          NO          -
degree          TEXT          NO          -
field_of_study  TEXT          YES         -
location        TEXT          YES         -
start_date      DATE          NO          -
end_date        DATE          YES         -
description     TEXT          YES         -
is_published    BOOLEAN       NO          FALSE
created_at      TIMESTAMPTZ   NO          NOW()
updated_at      TIMESTAMPTZ   NO          NOW()
```

**Primary Key:** `id`

**Unique Constraints:** none

**Foreign Keys:** none

**Check Constraints:**

- `institution <> ''`
- `degree <> ''`
- `field_of_study IS NULL OR field_of_study <> ''`
- `location IS NULL OR location <> ''`
- `description IS NULL OR description <> ''`
- `end_date IS NULL OR end_date >= start_date`

**Indexes:** PK only

**Delete Behavior:** standalone — hard delete removes the row

**Not present:** `display_order`, `is_featured`

**Intended list order (ties):**

```text
ORDER BY start_date DESC, id ASC
```

`id` is the deterministic tie-breaker when two entries share a `start_date`. No extra column required.

---

### 3.7 `certifications`

```text
Table: certifications

Column          Type          Null?       Default
--------------------------------------------------------
id              UUID          NO          gen_random_uuid()
name            TEXT          NO          -
issuer          TEXT          NO          -
credential_id   TEXT          YES         -
credential_url  TEXT          YES         -
issued_on       DATE          NO          -
expires_on      DATE          YES         -
is_published    BOOLEAN       NO          FALSE
display_order   INTEGER       NO          0
created_at      TIMESTAMPTZ   NO          NOW()
updated_at      TIMESTAMPTZ   NO          NOW()
```

**Primary Key:** `id`

**Unique Constraints:** none (same cert name from different issuers is possible)

**Foreign Keys:** none

**Check Constraints:**

- `name <> ''`
- `issuer <> ''`
- `credential_id IS NULL OR credential_id <> ''`
- `credential_url IS NULL OR credential_url <> ''`
- Expiry integrity:

```text
expires_on IS NULL OR expires_on >= issued_on
```

- `display_order >= 0`

**Indexes:** PK only

**Delete Behavior:** standalone hard delete

**Intended list order:** `ORDER BY display_order ASC, issued_on DESC`

---

### 3.8 `achievements`

```text
Table: achievements

Column          Type          Null?       Default
--------------------------------------------------------
id              UUID          NO          gen_random_uuid()
title           TEXT          NO          -
description     TEXT          YES         -
achieved_on     DATE          YES         -
url             TEXT          YES         -
is_published    BOOLEAN       NO          FALSE
display_order   INTEGER       NO          0
created_at      TIMESTAMPTZ   NO          NOW()
updated_at      TIMESTAMPTZ   NO          NOW()
```

**Primary Key:** `id`

**Unique Constraints:** none

**Foreign Keys:** none

**Check Constraints:**

- `title <> ''`
- `description IS NULL OR description <> ''`
- `url IS NULL OR url <> ''`
- `display_order >= 0`

**Indexes:** PK only

**Delete Behavior:** standalone hard delete

**Not present:** `is_featured`

**Intended list order:** `ORDER BY display_order ASC, achieved_on DESC NULLS LAST`

---

### 3.9 `social_platforms`

Lookup catalog for social / presence platforms. **Email is not a platform** (lives on `profile.email_public`).

Chosen as a table (not `TEXT + CHECK` on links) so platforms can carry metadata (label, default logo, order) and evolve without schema CHECK rewrites.

```text
Table: social_platforms

Column          Type          Null?       Default
--------------------------------------------------------
id              UUID          NO          gen_random_uuid()
slug            TEXT          NO          -
name            TEXT          NO          -
logo_url        TEXT          YES         -
display_order   INTEGER       NO          0
created_at      TIMESTAMPTZ   NO          NOW()
updated_at      TIMESTAMPTZ   NO          NOW()
```

**Primary Key:** `id`

**Unique Constraints:**

- `UNIQUE (slug)` — e.g. `github`, `linkedin`, `x`, `youtube`
- `UNIQUE (name)`

**Foreign Keys:** none

**Check Constraints:**

- `slug <> ''`
- `name <> ''`
- `logo_url IS NULL OR logo_url <> ''`
- Explicitly exclude email as a platform identity:

```text
slug <> 'email'
```

(Defense in depth; product rule is also “don’t seed email”.)

- `display_order >= 0`

**Indexes:** PK + unique indexes from UNIQUE

**Delete Behavior:**

- Referenced by `social_links.platform_id` → **`ON DELETE RESTRICT`**
- Cascades to: **nothing**

**Seed expectation (not part of DDL):** insert known platforms before links.

**Intended list order:** `ORDER BY display_order ASC, name ASC`

---

### 3.10 `social_links`

One outbound link per platform for the portfolio owner.

```text
Table: social_links

Column          Type          Null?       Default
--------------------------------------------------------
id              UUID          NO          gen_random_uuid()
platform_id     UUID          NO          -
label           TEXT          YES         -
url             TEXT          NO          -
logo_url        TEXT          YES         -
is_published    BOOLEAN       NO          FALSE
display_order   INTEGER       NO          0
created_at      TIMESTAMPTZ   NO          NOW()
updated_at      TIMESTAMPTZ   NO          NOW()
```

**Primary Key:** `id`

**Unique Constraints:**

- `UNIQUE (platform_id)` — at most one link per platform

**Foreign Keys:**

| Column | References | ON DELETE |
|--------|------------|-----------|
| `platform_id` | `social_platforms(id)` | **RESTRICT** |

**Check Constraints:**

- `url <> ''`
- `label IS NULL OR label <> ''`
- `logo_url IS NULL OR logo_url <> ''`
- `display_order >= 0`

**Indexes:**

- PK
- Unique on `platform_id` (covers FK lookups by platform)

**Delete Behavior:**

- Deleting a link removes only that row
- Cannot delete a platform while a link references it

**Notes:**

- `logo_url` on the link overrides platform default logo when set; otherwise UI may fall back to `social_platforms.logo_url`
- No email platform / email rows

**Intended list order:** `ORDER BY display_order ASC` (optionally join platform name)

---

### 3.11 `contact_messages`

Visitor-generated application data. Independent of `profile` and chat.

```text
Table: contact_messages

Column          Type          Null?       Default
--------------------------------------------------------
id              UUID          NO          gen_random_uuid()
name            TEXT          NO          -
email           TEXT          NO          -
subject         TEXT          YES         -
body            TEXT          NO          -
created_at      TIMESTAMPTZ   NO          NOW()
```

**Primary Key:** `id`

**Unique Constraints:** none

**Foreign Keys:** none — visitor `email` is **not** an FK to `profile`

**Check Constraints:**

- `name <> ''`
- `email <> ''`
- `body <> ''`
- `subject IS NULL OR subject <> ''`

**Indexes:**

- PK
- **`INDEX (created_at DESC)`** — admin inbox: newest first

**Why not a partial “unread” index?**

- `is_read` was **dropped** for v1 — no unread filter to optimize

**Delete Behavior:** standalone hard delete (spam / GDPR-style erasure)

**Not present:** `updated_at`, `is_read`

---

### 3.12 `chat_sessions`

Anonymous application session for the AI representative. Not a visitor identity / CRM record.

```text
Table: chat_sessions

Column          Type          Null?       Default
--------------------------------------------------------
id              UUID          NO          gen_random_uuid()
visitor_key     TEXT          NO          -
started_at      TIMESTAMPTZ   NO          NOW()
last_message_at TIMESTAMPTZ   NO          NOW()
ended_at        TIMESTAMPTZ   YES         -
metadata        JSONB         YES         -
```

**Primary Key:** `id`

**Unique Constraints:** none (`visitor_key` may appear on many sessions over time)

**Foreign Keys:** none

**Check Constraints:**

- `visitor_key <> ''`
- `last_message_at >= started_at`
- `ended_at IS NULL OR ended_at >= started_at`

**Indexes:**

- PK
- **`INDEX (visitor_key)`** — resume / list sessions for an anonymous client key
- **`INDEX (last_message_at DESC)`** — operational “recent conversations”

**Delete Behavior:**

- Deleting a session **CASCADE**s to `chat_messages`

**`visitor_key` semantics:**

| Is | Is not |
|----|--------|
| Opaque client-generated token (UUID string or similar) | Login identity |
| Cookie / localStorage correlation handle | Email, phone, IP-as-PK |
| `TEXT NOT NULL` | FK to a people table |

Prefer generating a UUID in the client/API and storing its string form — type stays `TEXT` so the DB does not imply “this is a person UUID.”

**`metadata`:** optional JSONB for locale / coarse UA class / client version. Keep minimal. Reject stuffing portfolio content here.

**`ended_at`:** nullable; `NULL` = open / not explicitly closed.

**Not present:** `updated_at` (use `last_message_at` / `ended_at` as real event times)

---

### 3.13 `chat_messages`

```text
Table: chat_messages

Column          Type          Null?       Default
--------------------------------------------------------
id              UUID          NO          gen_random_uuid()
session_id      UUID          NO          -
role            TEXT          NO          -
content         TEXT          NO          -
token_count     INTEGER       YES         -
created_at      TIMESTAMPTZ   NO          NOW()
```

**Primary Key:** `id`

**Unique Constraints:** none

**Foreign Keys:**

| Column | References | ON DELETE |
|--------|------------|-----------|
| `session_id` | `chat_sessions(id)` | **CASCADE** |

**Check Constraints:**

- `role IN ('user', 'assistant', 'system')`
- `content <> ''`
- `token_count IS NULL OR token_count >= 0`

**Indexes:**

- PK
- **`INDEX (session_id, created_at ASC)`** — load a thread in chronological order (covers FK + history query)

**Delete Behavior:** removed when parent session is deleted (`CASCADE`)

**Intended order:** `ORDER BY created_at ASC, id ASC`

**Not present:** `updated_at` (messages are append-only)

---

### 3.14 `project_skills`

Junction: project ↔ skill.

```text
Table: project_skills

Column          Type          Null?       Default
--------------------------------------------------------
project_id      UUID          NO          -
skill_id        UUID          NO          -
display_order   INTEGER       NO          0
```

**Primary Key:** `(project_id, skill_id)` — prevents duplicate membership

**Unique Constraints:** covered by PK

**Foreign Keys:**

| Column | References | ON DELETE |
|--------|------------|-----------|
| `project_id` | `projects(id)` | **CASCADE** |
| `skill_id` | `skills(id)` | **CASCADE** |

**Check Constraints:**

- `display_order >= 0`

**Indexes:**

- PK `(project_id, skill_id)` — efficient for “skills for this project”
- **`INDEX (skill_id)`** — reverse: “projects using this skill” / AI queries

**Delete Behavior:**

| Delete | Effect |
|--------|--------|
| Project | Join rows cascade away |
| Skill | Join rows cascade away; **projects remain** |

No timestamps — pure association.

---

### 3.15 `experience_skills`

Junction: experience ↔ skill.

```text
Table: experience_skills

Column          Type          Null?       Default
--------------------------------------------------------
experience_id   UUID          NO          -
skill_id        UUID          NO          -
display_order   INTEGER       NO          0
```

**Primary Key:** `(experience_id, skill_id)`

**Check Constraints:**

- `display_order >= 0`

**Foreign Keys:**

| Column | References | ON DELETE |
|--------|------------|-----------|
| `experience_id` | `experience(id)` | **CASCADE** |
| `skill_id` | `skills(id)` | **CASCADE** |

**Indexes:**

- PK `(experience_id, skill_id)` — “skills for this role”
- **`INDEX (skill_id)`** — “where was this skill used professionally?”

**Delete Behavior:** same pattern as `project_skills` (cascade joins only)

---

## 4. Relationship matrix

| Parent | Child | Cardinality | Implementing FK | ON DELETE |
|--------|-------|-------------|-----------------|-----------|
| *(singleton)* | `profile` | 0..1 row total | `lock_key` UNIQUE+CHECK | n/a |
| `skill_categories` | `skills` | 1:N | `skills.category_id` | **RESTRICT** |
| `projects` | `project_skills` | 1:N | `project_skills.project_id` | **CASCADE** |
| `skills` | `project_skills` | 1:N | `project_skills.skill_id` | **CASCADE** |
| `experience` | `experience_skills` | 1:N | `experience_skills.experience_id` | **CASCADE** |
| `skills` | `experience_skills` | 1:N | `experience_skills.skill_id` | **CASCADE** |
| `projects` ↔ `skills` | via `project_skills` | N:M | composite | see above |
| `experience` ↔ `skills` | via `experience_skills` | N:M | composite | see above |
| `social_platforms` | `social_links` | 1:N | `social_links.platform_id` | **RESTRICT** |
| `chat_sessions` | `chat_messages` | 1:N | `chat_messages.session_id` | **CASCADE** |

### Explicit non-relationships

| From | To | Notes |
|------|----|-------|
| Chat | Projects / content | AI reads via services, not FKs |
| ContactMessage | Profile / Chat | Independent visitor data |
| Education | Skills | No join in v1 |
| SocialLink | Email | Email on `profile` only |

---

## 5. Index strategy summary

| Table | Index | Reason |
|-------|-------|--------|
| `profile` | `UNIQUE (lock_key)` | Singleton |
| `skill_categories` | `UNIQUE (slug)`, `UNIQUE (name)` | Identity |
| `skills` | `UNIQUE (slug)`, `UNIQUE (name)` | Identity |
| `skills` | `(category_id)` | List by category |
| `projects` | `UNIQUE (slug)` | Public deep link lookup |
| `social_platforms` | `UNIQUE (slug)`, `UNIQUE (name)` | Catalog identity |
| `social_links` | `UNIQUE (platform_id)` | One link per platform + FK |
| `contact_messages` | `(created_at DESC)` | Inbox newest-first |
| `chat_sessions` | `(visitor_key)` | Resume anonymous sessions |
| `chat_sessions` | `(last_message_at DESC)` | Recent session ops |
| `chat_messages` | `(session_id, created_at ASC)` | Thread load + FK support |
| `project_skills` | `PRIMARY KEY (project_id, skill_id)` | Membership + forward lookup |
| `project_skills` | `(skill_id)` | Reverse: projects for skill |
| `experience_skills` | `PRIMARY KEY (experience_id, skill_id)` | Membership + forward lookup |
| `experience_skills` | `(skill_id)` | Reverse: experience for skill |

**Explicitly not indexed in v1:**

- Extra non-unique index on any column already covered by UNIQUE/PK
- `(is_published, display_order)` composites — defer until needed
- Full-text on descriptions — later product feature

---

## 6. What may be deleted?

| Entity | Typical delete? | Cascades to | Must not destroy |
|--------|-----------------|-------------|------------------|
| `profile` | Rare / avoid | — | — |
| `skill_categories` | Only if unused | blocked if skills exist | skills |
| `skills` | Yes, with care | join rows | projects, experience |
| `projects` | Yes | `project_skills` | skills |
| `experience` | Yes | `experience_skills` | skills |
| `education` / certs / achievements | Yes | — | — |
| `social_platforms` | Only if unused | nothing (RESTRICT) | social_links |
| `social_links` | Yes | — | platforms |
| `contact_messages` | Yes (ops / privacy) | — | — |
| `chat_sessions` | Yes | `chat_messages` | — |
| `chat_messages` | Via session or direct | — | — |

Soft-hide for portfolio content uses `is_published = FALSE` instead of delete whenever possible.

---

## 7. Delta from `domain-model.md`

| Topic | Domain model | DDL design |
|-------|--------------|------------|
| Social platform vocabulary | TEXT platform key + CHECK lean | **`social_platforms` table** (user decision) |
| `SocialLink.platform` | text key | **`platform_id` FK** |
| Profile singleton | Conceptual | **`lock_key` BOOLEAN + CHECK + UNIQUE** |
| Join `display_order` | Optional | **`INTEGER NOT NULL DEFAULT 0`** + `CHECK (>= 0)` |
| Empty strings | Implied invalid | Explicit `<> ''` CHECKs |
| Skill / category name uniqueness | Slug unique | **Also `UNIQUE (name)`** |

After DDL approval, both docs should remain aligned (`social_platforms` already reflected in `domain-model.md`).

---

## 8. Out of scope for `000002`

- Seed data (platforms, profile row, sample projects)
- `updated_at` maintenance triggers
- Dropping `schema_test` from `000001` (follow-up migration)
- Admin users / auth tables
- Full-text search
- Media / asset storage tables

---

## 9. Review checklist

- [x] Singleton `profile` accepted (`lock_key` approach)
- [x] `social_platforms` + `social_links` accepted
- [x] Every FK has explicit ON DELETE; **no ON UPDATE CASCADE**
- [x] `display_order >= 0` on all ordered tables
- [x] `social_platforms` delete docs match RESTRICT (no silent cascade)
- [x] Reverse indexes on junction `skill_id` accepted
- [x] No duplicate indexes on UNIQUE columns
- [x] No `is_read`, no skill `proficiency`, no education `display_order`
- [x] `is_published DEFAULT FALSE`
- [x] Chat `role` / project vocabularies via TEXT+CHECK
- [x] Creation / drop order documented
- [x] Write `000002_portfolio_schema.up.sql` / `.down.sql`
```

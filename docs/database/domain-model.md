# Domain Model — Portfolio Backend

**Task:** `003` — Database Schema & Domain Modeling  
**Branch:** `task/003_Schema_and_Models`  
**Status:** Design for review (no SQL migration yet)

This document answers:

> What data does the portfolio backend own, how is that data conceptually represented, and how are the entities related?

It deliberately stops before PostgreSQL DDL. Once this model is approved, it becomes migration `000002_portfolio_schema`.

---

## 1. Design principles

1. **Content vs behavior** — portfolio content and application/runtime data stay in separate conceptual groups.
2. **Entities earn their table** — something is an entity only if it has identity, is queried independently, or participates in relationships.
3. **Attributes stay attributes** — a company name on an experience row is not a `companies` table unless we need shared company records.
4. **Stable IDs + selective slugs** — UUID primary keys for persistence. **Project `slug`** is the public deep-link contract (`/projects/...`). **Skill `slug`** may exist for internal stability/admin use but is **not** a public deep-link contract in v1.
5. **Presentation order ≠ creation time** — curated lists use explicit `display_order` unless a natural date order is enough (e.g. Education → `start_date`).
6. **Soft visibility where useful** — publish/hide without delete; `is_published` defaults to **`FALSE`**.
7. **No premature JSONB** — structured columns first; JSONB only if data is genuinely semi-structured.
8. **AI reads through services later** — schema should expose clean portfolio aggregates; chat storage is separate application data.
9. **Don't invent admin/auth yet** — no `users` / `admin_users` until content management needs them.
10. **Drop test artifacts before production freeze** — `schema_test` from `000001` is infrastructure validation only; remove before schema is considered stable.

---

## 2. Two data worlds

```text
                         PostgreSQL (conceptual)
                                  │
               ┌──────────────────┴──────────────────┐
               │                                     │
        Portfolio Content                     Application Data
               │                                     │
    What the site displays /              What visitors / AI
    what the AI may summarize             produce at runtime
               │                                     │
    profile, projects, skills,            contact_messages,
    experience, education, …              chat_sessions,
                                          chat_messages
```

| World | Owned by | Mutated by | Examples |
|-------|----------|------------|----------|
| Portfolio content | Site owner (you) | Admin/seed/migration | Projects, skills, experience |
| Application data | System + visitors | API at runtime | Contact form, chat history |

The AI representative **reads** portfolio content (via a future Portfolio Service) and **writes** conversation records into application data. It should not treat chat tables and project tables as one interchangeable bag of rows.

---

## 3. Entity inventory (proposed)

### Included now

| Entity | World | Why it exists |
|--------|-------|---------------|
| `Profile` | Content | Singleton site identity (name, headline, bio, etc.) |
| `Project` | Content | First-class portfolio work |
| `SkillCategory` | Content | Constrained vocabulary for grouping skills |
| `Skill` | Content | Shared vocabulary for skills section, project tech, AI |
| `ProjectSkill` | Content | Many-to-many link: project ↔ skill |
| `Experience` | Content | Professional history entries |
| `ExperienceSkill` | Content | Many-to-many link: experience ↔ skill |
| `Education` | Content | Academic / formal education entries |
| `Certification` | Content | Credentials with issuer + dates |
| `Achievement` | Content | Awards, highlights, notable wins |
| `SocialPlatform` | Content | Catalog of allowed social / presence platforms |
| `SocialLink` | Content | Public profile links (GitHub, LinkedIn, …) |
| `ContactMessage` | Application | Inbound contact-form submissions |
| `ChatSession` | Application | One visitor ↔ AI conversation thread |
| `ChatMessage` | Application | Individual messages inside a session |

### Deferred (not in v1 schema)

| Candidate | Why defer |
|-----------|-----------|
| `User` / `AdminUser` | No authenticated content CMS yet; seeding/migrations can populate content |
| `Company` / `School` | Names are attributes on experience/education; no shared company catalog needed |
| `ProjectImage` / media gallery | Start with one cover/image URL on `Project`; expand if needed |
| `Tag` generic taxonomy | Skills already cover technology tagging; avoid parallel taxonomies |
| `Resume` / versioned CV blob | Out of scope; content lives in structured tables |

### Rejected as standalone entities

| Idea | Decision |
|------|----------|
| “Identity” as multi-row people table | One portfolio owner → one `Profile` row (singleton), not a people directory |
| Generic `content` / `metadata` / `attributes` tables | Over-flexible; hides structure the API and AI need |
| Storing entire projects as one JSONB document | Relationships and constraints matter; use columns |
| Email as a `SocialLink` platform | Contact identity lives on `Profile.email_public` (and inbound `ContactMessage.email`), not as a social platform row |
| `is_featured` on Experience / Education / Achievement | Not in v1 unless a UI later requires highlighting those sections |

---

## 4. Entities in detail

### 4.1 Profile *(portfolio content)*

Represents the **site owner identity** shown across the portfolio (hero, about, AI grounding).

This is a **singleton** in practice: the backend owns one profile. We still model it as a table (not config) so bio/headline can change without redeploying, and so the AI can load a stable “about me” record.

**Properties**

| Property | Notes |
|----------|--------|
| `id` | UUID |
| `full_name` | Display name |
| `headline` | Short professional tagline |
| `summary` | Longer about / bio |
| `location` | Optional free-text location |
| `email_public` | Optional publicly displayed email (this is where email lives — not `SocialLink`) |
| `avatar_url` | Optional image URL |
| `created_at` / `updated_at` | TIMESTAMPTZ |

**Not included**

- Password / auth fields — not a login account
- Multiple profiles — out of scope

**Relationships**

- None required. Other content does **not** FK to profile in v1 (implicit single owner).

---

### 4.2 Project *(portfolio content)*

Represents a project displayed on the portfolio (and available to the AI as project knowledge).

**Properties**

| Property | Notes |
|----------|--------|
| `id` | UUID PK |
| `slug` | Unique, URL-safe public deep link (`/projects/portfolio-website`) |
| `title` | Required |
| `summary` | Short card/blurb text |
| `description` | Full write-up as **rich markdown** (`TEXT`); rendering is an app concern |
| `role` | Your role on the project, e.g. `solo`, `lead`, `contributor`, `maintainer` |
| `project_type` | Project shape, e.g. `solo`, `team`, `open_source`, `client`, `academic` |
| `repo_url` | Optional |
| `live_url` | Optional |
| `image_url` | Optional cover image |
| `started_on` | Optional date (calendar date, not timestamptz) |
| `ended_on` | Optional; null = ongoing |
| `is_featured` | Highlight on home / featured section |
| `is_published` | Hide without deleting; **DEFAULT FALSE** |
| `display_order` | Explicit presentation order |
| `created_at` / `updated_at` | TIMESTAMPTZ |

**Relationships**

- Many-to-many with `Skill` via `ProjectSkill`

**Constraint intent**

- `role` and `project_type` constrained to allowed values (CHECK or app enum)

---

### 4.3 SkillCategory *(portfolio content)*

Constrained grouping for skills (e.g. Languages, Frameworks, Tools, Cloud).

A free-text `category` column on `Skill` would drift (`Backend`, `backend`, `Back-end`). A small lookup table keeps groupings consistent for UI sections and AI answers.

**Properties**

| Property | Notes |
|----------|--------|
| `id` | UUID PK |
| `slug` | Unique internal key (`language`, `framework`, `tool`, `cloud`, `other`) |
| `name` | Display label (`Languages`, `Frameworks`) |
| `display_order` | Order of category sections in the Skills UI |
| `created_at` / `updated_at` | TIMESTAMPTZ |

**Relationships**

- One-to-many → `Skill`

**Delete behavior (intended)**

- Prefer `ON DELETE RESTRICT` while skills still reference the category (prevent accidental orphaning of groupings).

---

### 4.4 Skill *(portfolio content)*

First-class skill because it serves **multiple surfaces**:

- Skills section
- Project technologies
- Skills used in a professional role (experience)
- Filtering / AI answers (“what have you built with Go?”, “where did you use PostgreSQL?”)

**Properties**

| Property | Notes |
|----------|--------|
| `id` | UUID PK |
| `slug` | Unique **internal** stable key (`go`, `react`, `postgresql`) — useful for seeds/admin/idempotency; **not** a public deep-link contract in v1 |
| `name` | Display label (`Go`, `React`) |
| `category_id` | FK → `SkillCategory` (required) |
| `logo_url` | Optional skill icon / logo image URL |
| `is_featured` | Show in primary skills strip |
| `is_published` | Soft visibility; **DEFAULT FALSE** |
| `display_order` | Order within its category / skills section |
| `created_at` / `updated_at` | TIMESTAMPTZ |

**Not in v1**

- `proficiency` — dropped to avoid fake precision / vanity rankings

**Why not free-text tags on projects only?**

That duplicates “React” across projects, prevents a clean Skills section, and makes AI queries messier. A shared `Skill` entity owns the canonical name once.

**Relationships**

- Many-to-one → `SkillCategory`
- Many-to-many with `Project` via `ProjectSkill`
- Many-to-many with `Experience` via `ExperienceSkill`

---

### 4.5 ProjectSkill *(join — portfolio content)*

Represents membership of a skill on a project.

**Properties**

| Property | Notes |
|----------|--------|
| `project_id` | FK → Project |
| `skill_id` | FK → Skill |
| `display_order` | Optional order of tech badges on a project |

**Identity**

- Composite primary key `(project_id, skill_id)` is enough; no separate UUID required unless we later attach metadata.

**Delete behavior (intended)**

- Deleting a project → remove its join rows (`CASCADE`)
- Deleting a skill → remove join rows (`CASCADE`) — or restrict delete if skill is still referenced (product choice; prefer CASCADE for simplicity in v1)

---

### 4.6 Experience *(portfolio content)*

Represents one professional role / employment entry.

**Properties**

| Property | Notes |
|----------|--------|
| `id` | UUID PK |
| `company` | Text attribute (not a Company entity) |
| `title` | Role title |
| `location` | Optional |
| `logo_url` | Optional company / org logo image URL |
| `employment_type` | Optional: `full_time`, `part_time`, `contract`, `internship`, … |
| `start_date` | Required date |
| `end_date` | Null = current role |
| `description` | Bullet narrative / summary (rich markdown) |
| `is_published` | Soft visibility; **DEFAULT FALSE** |
| `display_order` | Presentation order (usually reverse-chronological, but explicit) |
| `created_at` / `updated_at` | TIMESTAMPTZ |

**Not in v1**

- `is_featured` — not added unless the UI later requires featuring specific roles

**Relationships**

- Many-to-many with `Skill` via `ExperienceSkill`

**Constraint intent**

- If `end_date` is set, it should be `>= start_date` (CHECK)

---

### 4.7 ExperienceSkill *(join — portfolio content)*

Represents membership of a skill on an experience entry (“skills used in this role”).

Same canonical `Skill` rows as projects — one vocabulary for the whole portfolio.

**Properties**

| Property | Notes |
|----------|--------|
| `experience_id` | FK → Experience |
| `skill_id` | FK → Skill |
| `display_order` | Optional order of skill badges on an experience entry |

**Identity**

- Composite primary key `(experience_id, skill_id)`; no separate UUID.

**Delete behavior (intended)**

- Deleting an experience → remove its join rows (`CASCADE`)
- Deleting a skill → remove join rows (`CASCADE`)

---

### 4.8 Education *(portfolio content)*

Represents one education entry.

**Properties**

| Property | Notes |
|----------|--------|
| `id` | UUID PK |
| `institution` | School / university name (attribute) |
| `degree` | e.g. B.Tech, M.Sc |
| `field_of_study` | Optional |
| `location` | Optional |
| `start_date` | Required — used for list ordering (`ORDER BY start_date DESC`) |
| `end_date` | Null = in progress |
| `description` | Optional notes, honors |
| `is_published` | Soft visibility; **DEFAULT FALSE** |
| `created_at` / `updated_at` | TIMESTAMPTZ |

**Not in v1**

- `display_order` — sort by `start_date` instead
- `is_featured` — not added unless the UI later requires it

**Relationships**

- None in v1

---

### 4.9 Certification *(portfolio content)*

Represents a credential.

**Properties**

| Property | Notes |
|----------|--------|
| `id` | UUID PK |
| `name` | Certificate title |
| `issuer` | Issuing org |
| `credential_id` | Optional external ID |
| `credential_url` | Optional verification link |
| `issued_on` | Date |
| `expires_on` | Optional |
| `is_published` | Soft visibility; **DEFAULT FALSE** |
| `display_order` | Presentation order |
| `created_at` / `updated_at` | TIMESTAMPTZ |

**Relationships**

- None in v1

---

### 4.10 Achievement *(portfolio content)*

Represents awards, competitions, notable non-job highlights that aren't certifications or projects.

**Properties**

| Property | Notes |
|----------|--------|
| `id` | UUID PK |
| `title` | Required |
| `description` | Optional |
| `achieved_on` | Optional date |
| `url` | Optional proof / announcement link |
| `is_published` | Soft visibility; **DEFAULT FALSE** |
| `display_order` | Presentation order |
| `created_at` / `updated_at` | TIMESTAMPTZ |

**Not in v1**

- `is_featured` — not added unless the UI later requires it

**Why keep separate from Project / Certification?**

Different presentation semantics: achievements are often timeline highlights, not installable credentials or buildable projects. Merging them into a generic “highlights” table loses clarity for the AI and the UI.

---

### 4.11 SocialPlatform *(portfolio content)*

Catalog of allowed social / presence platforms (metadata: slug, name, default logo, order).

Email is **not** a platform — see `Profile.email_public`.

**Properties**

| Property | Notes |
|----------|--------|
| `id` | UUID PK |
| `slug` | Unique internal key (`github`, `linkedin`, `x`, `youtube`) — must not be `email` |
| `name` | Display label |
| `logo_url` | Optional default platform icon |
| `display_order` | Catalog / UI ordering |
| `created_at` / `updated_at` | TIMESTAMPTZ |

**Relationships**

- One-to-many → `SocialLink`

**Delete behavior (intended)**

- `ON DELETE RESTRICT` while links still reference the platform

---

### 4.12 SocialLink *(portfolio content)*

Represents one outbound social / presence link.

**Properties**

| Property | Notes |
|----------|--------|
| `id` | UUID PK |
| `platform_id` | FK → `SocialPlatform` (required); unique — one link per platform |
| `label` | Optional display override |
| `url` | Required |
| `logo_url` | Optional override of platform default icon |
| `is_published` | Soft visibility; **DEFAULT FALSE** |
| `display_order` | Icon/link order in footer/header |
| `created_at` / `updated_at` | TIMESTAMPTZ |

**Not a SocialLink**

- **Email** — not a platform. Public email belongs on `Profile.email_public`; contact-form email is `ContactMessage.email`.

**Why a platforms table?**

Platforms need shared metadata (name, default logo, order). A lookup table is clearer than rewriting CHECK constraints when platforms are added.

---

### 4.13 ContactMessage *(application data)*

Inbound message from the contact form.

**Properties**

| Property | Notes |
|----------|--------|
| `id` | UUID PK |
| `name` | Sender name |
| `email` | Sender email |
| `subject` | Optional |
| `body` | Required |
| `created_at` | TIMESTAMPTZ (immutable in practice) |

**Not in v1**

- `is_read` — not needed yet; add later only if admin triage UI requires it

**Notes**

- No `updated_at` required unless you edit messages (unlikely).
- No FK to chat — contact form ≠ AI conversation.
- Do not expose these rows to the AI representative.

---

### 4.14 ChatSession *(application data)*

One conversation thread between a visitor and the AI representative.

**Properties**

| Property | Notes |
|----------|--------|
| `id` | UUID PK |
| `visitor_key` | Opaque client/session key (cookie/device), **not** PII by itself |
| `started_at` | TIMESTAMPTZ |
| `last_message_at` | TIMESTAMPTZ, updated as messages arrive |
| `ended_at` | Nullable TIMESTAMPTZ — null means the session is still open / not explicitly closed; keep nullable |
| `metadata` | Optional **limited** JSONB for non-relational session hints (user agent class, locale) — keep minimal |

**Why allow optional JSONB here?**

Session telemetry is semi-structured and product-specific. Portfolio content tables should stay relational; a small metadata bag on sessions is the only place JSONB is tentatively justified.

**Relationships**

- One-to-many → `ChatMessage`

---

### 4.15 ChatMessage *(application data)*

One message in a chat session.

**Properties**

| Property | Notes |
|----------|--------|
| `id` | UUID PK |
| `session_id` | FK → ChatSession |
| `role` | `user` \| `assistant` \| `system` (CHECK) |
| `content` | Message text |
| `created_at` | TIMESTAMPTZ |
| `token_count` | Optional, for future cost/observability |

**Delete behavior (intended)**

- Deleting a session cascades to its messages.

**Ordering**

- Chronological via `created_at` (and `id` as tie-breaker) is correct here — messages are an event log, not curated portfolio order.

---

## 5. Relationship model

```text
Profile                 (singleton, no FKs in v1)

SkillCategory ──── 1:N ──── Skills
                              │
              ┌───────────────┼───────────────┐
              │               │               │
        many-to-many    (canonical)     many-to-many
              │               │               │
         Projects             │          Experience
              │               │               │
        ProjectSkill          │      ExperienceSkill


Education               (standalone content)
Certification           (standalone content)
Achievement             (standalone content)

SocialPlatform ──── 1:N ──── SocialLink


ContactMessage          (standalone application data)


ChatSession ──── 1:N ──── ChatMessage
```

### Cardinalities

| From | To | Type | Implementation |
|------|----|------|----------------|
| SkillCategory | Skill | one-to-many | `skills.category_id` FK |
| SocialPlatform | SocialLink | one-to-many | `social_links.platform_id` FK |
| Project | Skill | many-to-many | `project_skills` join table |
| Experience | Skill | many-to-many | `experience_skills` join table |
| ChatSession | ChatMessage | one-to-many | `chat_messages.session_id` FK |
| Everything else | — | independent rows | no FK in v1 |

### Explicit non-relationships (for now)

- Education ↛ Skill
- Project ↛ Experience
- Chat ↛ Project (AI retrieves portfolio via service queries, not FKs from messages)
- ContactMessage ↛ ChatSession

---

## 6. Identifiers

| Kind | Use |
|------|-----|
| `UUID` PK | All persistent entities (`gen_random_uuid()`) |
| `Project.slug` | **Public** deep-link contract (`/projects/...`) |
| `Skill.slug` / `SkillCategory.slug` | **Internal** stable keys (unique); not treated as public URL contracts in v1 |
| Composite PK | `ProjectSkill (project_id, skill_id)`, `ExperienceSkill (experience_id, skill_id)` |
| No slug | Experience, education, etc. (list sections, not deep-linked resources initially) |

If experience/education later need shareable deep links, add `slug` then — not before. If skills later need public pages, promote `Skill.slug` to a public contract then.

---

## 7. Cross-cutting fields

### Timestamps

- Application timestamps: `TIMESTAMPTZ`
- Calendar career dates (`start_date`, `issued_on`): `DATE` (no timezone semantics)
- `updated_at` is **not** auto-maintained by PostgreSQL unless we add a trigger or set it in the repository layer later

### Visibility

Use `is_published BOOLEAN NOT NULL DEFAULT FALSE` on public portfolio content:

- Project, Skill, Experience, Education, Certification, Achievement, SocialLink

Do **not** put `is_published` on ContactMessage / Chat* — those aren't “published content.”

Seeded demos may explicitly set `TRUE` in seed data; the column default remains `FALSE`.

### Ordering

- Use `display_order INTEGER NOT NULL` on curated lists where presentation order is independent of dates (projects, skills, skill categories, experience, certifications, achievements, social links).
- **Education** has no `display_order` — sort by `start_date` (typically `DESC`).
- Do not use `ORDER BY created_at` for portfolio presentation.

### Featured flags

`is_featured` exists only where the UI already needs a highlight strip:

- **Yes in v1:** `Project`, `Skill`
- **Not in v1:** `Experience`, `Education`, `Achievement` (add only if UI requires it)

Prefer `is_published` / `is_featured` over a generic `status` text field unless a real multi-state workflow appears (draft → review → published).

---

## 8. Constraints & indexes (intent only — not SQL yet)

### Constraints we expect to encode later

| Area | Intent |
|------|--------|
| PKs | UUID (or composite for join) |
| UNIQUE | `projects.slug`, `skills.slug`, `skill_categories.slug`, likely `social_links.platform` |
| NOT NULL | Core display fields (title, name, urls where required); `skills.category_id` |
| CHECK | `end_date >= start_date` where both present; chat `role` in allowed set; project `role` / `project_type` in allowed sets |
| FK | `skills.category_id` → skill_categories; `project_skills` → projects/skills; `experience_skills` → experience/skills; `chat_messages.session_id` → chat_sessions |
| ON DELETE | Join rows CASCADE with parent; chat messages CASCADE with session; skill category delete RESTRICT while referenced |

### Indexes — deliberate restraint

| Index | Needed? |
|-------|---------|
| Unique on `slug` | Yes — uniqueness **implies** a unique index; do not add a second non-unique index on the same column |
| FK columns (`session_id`, join FKs) | Usually yes for join/delete performance |
| `(is_published, display_order)` | Maybe later if lists are large; not mandatory on day one for a personal portfolio |
| Full-text on descriptions | Later, if search is a product feature |

Rule of thumb for this task: **index what uniqueness or FKs require; defer speculative read indexes.**

---

## 9. Normalization notes (practical)

| Risk | How this model avoids it |
|------|--------------------------|
| Duplicated skill names across projects / experience | Canonical `Skill` + `ProjectSkill` / `ExperienceSkill` join tables |
| Update anomaly on “React” rename | Change one `skills.name` / `slug` |
| Company as entity with one row each | Keep `company` as text on Experience |
| Mixing chat history into project rows | Separate application tables |
| JSON blob for entire portfolio | Rejected |

We are not chasing textbook NF proofs; we are ensuring **clear ownership** of each fact.

---

## 10. AI representative implications

Future flow:

```text
Visitor → AI Representative → Portfolio Service → content tables
                           ↘ writes → chat_sessions / chat_messages
```

Schema consequences:

1. Content tables are readable, structured, and filterable (`is_published = true`).
2. Chat tables store conversation only — no FK coupling into projects.
3. Contact messages stay out of the AI’s data path.
4. `Profile` + published content form the AI’s grounded knowledge surface (via service methods, not raw SQL from the model layer later).

---

## 11. Migration plan (after approval)

1. Keep `000001_schema_test` as historical validation **for now**.
2. Add `000002_portfolio_schema.up.sql` / `.down.sql` implementing **this** model only.
3. Before calling the DB “production-ready,” add a follow-up migration (or squash strategy) that **drops `schema_test`** so test artifacts are not permanent schema.
4. Seed content is a separate concern (not part of domain approval).

---

## 12. Decisions locked for v1

| Topic | Decision |
|-------|----------|
| `Skill.proficiency` | **Dropped** |
| Skill grouping | **`SkillCategory` table**; `skills.category_id` FK |
| Email vs social | Email on **`Profile.email_public`** / contact messages — **not** a `SocialLink` platform |
| `is_published` default | **`FALSE`** |
| Project slug | **Public** deep-link contract |
| Skill slug | **Internal** unique key only (not a public URL contract yet) |
| `ContactMessage.is_read` | **Not in v1** |
| `ChatSession.ended_at` | **Keep, nullable** |
| `is_featured` on Experience / Education / Achievement | **Not in v1** |
| Social platforms | **`social_platforms` table**; one `social_links` row per platform |
| Profile singleton | Enforced in DB — see [`ddl-design.md`](./ddl-design.md) |
| Chat `metadata` | **Minimal JSONB** on `chat_sessions` |

## 13. Remaining open decisions

None blocking DDL design — review [`ddl-design.md`](./ddl-design.md) next.

---

## 14. Definition of done for this document

- [x] Entity inventory with include / defer / reject
- [x] Content vs application separation
- [x] Per-entity properties and purpose
- [x] Relationship diagram and cardinalities
- [x] Identifier / timestamp / visibility / ordering policy
- [x] Constraint & index *intent* (no DDL yet)
- [ ] Stakeholder review / approval of domain + DDL design ← **you are here**
- [ ] Then: `000002` up/down migrations and migrate/rollback verification
```

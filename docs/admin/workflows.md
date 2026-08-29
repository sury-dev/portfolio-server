# Admin Workflows

**Companion to:** [`architecture.md`](./architecture.md)  
**Status:** Design (revised) — singleton `admin` row, CLI password, single session

This document answers:

> How do you set the admin password, log in (killing old sessions), refresh, log out, and run CRUD?

---

## 1. One-time setup

### 1.1 Apply migrations

```text
000001 → 000002 → 000003 (admin table)
```

`000003` creates the singleton `admin` table (see architecture). Until `set-password` runs, login must fail (no row or empty hash).

### 1.2 Configure token infrastructure only

Password is **not** in config.

```text
[ADMIN]
JWT_ACCESS_SECRET = <random>       # if access tokens are JWTs
JWT_REFRESH_SECRET = <random>      # optional / omit for opaque refresh
ACCESS_TOKEN_TTL_SEC = 900
REFRESH_TOKEN_TTL_SEC = 604800
```

Database connection still comes from existing `[DATABASE]` config (CLI and API both use it).

### 1.3 Set password via Admin CLI

```text
go run ./cmd/admin set-password
# prompts for password (and confirmation)
```

Internal steps:

```text
1. Read password from prompt (preferred)
2. Hash with argon2id (or project-standard hasher)
3. UPSERT admin singleton:
     password_hash = <hash>
     clear access_* and refresh_* session columns
4. Success message without printing the hash
```

**Rules**

- Never write the password or hash into `config.conf`
- Never commit secrets
- Re-running `set-password` rotates the credential and **logs out** any active session

### 1.4 Verify setup

```text
go run ./cmd/admin status
```

Illustrative output (no secrets):

```text
admin row: present
password:  set
session:   inactive
```

---

## 2. Login workflow (overwrites previous session)

### Goal

Verify password against `admin.password_hash` and establish the **only** active session.

### Sequence

```text
Client                         Server                         admin row
  │                              │                                │
  │  POST /admin/auth/login      │                                │
  │  { "password": "..." }       │                                │
  │─────────────────────────────▶│                                │
  │                              │  SELECT singleton admin        │
  │                              │───────────────────────────────▶│
  │                              │  verify(password, hash)        │
  │                              │                                │
  │                              │  [fail] 401                    │
  │◀─────────────────────────────│                                │
  │                              │                                │
  │                              │  [ok] mint access + refresh    │
  │                              │  OVERWRITE token hashes/exp    │
  │                              │───────────────────────────────▶│
  │                              │  (previous session dead)       │
  │  200 { access, refresh, … }  │                                │
  │◀─────────────────────────────│                                │
```

### Important consequence

```text
Device A logged in
Device B logs in   →  Device A’s tokens stop working immediately
```

No “logout-all” endpoint is required for that behavior — **login is logout-all**.

### Request (illustrative)

```http
POST /admin/auth/login
Content-Type: application/json

{
  "password": "your-strong-password"
}
```

### Success response (illustrative)

```json
{
  "access_token": "<token>",
  "refresh_token": "<token>",
  "token_type": "Bearer",
  "expires_in": 900
}
```

### Failure cases

| Condition | HTTP |
|-----------|------|
| Admin row missing / password unset | 503 or 401 (prefer consistent 401 to clients; log server-side) |
| Wrong password | 401 |
| Empty body | 400 |
| Rate limited | 429 |

---

## 3. Authenticated request workflow

```text
Authorization: Bearer <access_token>
        │
        ▼
1. Parse / verify token structure (JWT sig if applicable)
2. Hash raw token
3. Load admin row
4. Require hash == access_token_hash
5. Require now < access_token_expires_at
        │
        ├── fail → 401
        └── ok   → CRUD handler
```

### Rules

- `/admin/auth/login` and `/admin/auth/refresh` are public among admin auth routes
- All other `/admin/*` routes require the **current** access token
- An old access token after a new login fails step 4 even if JWT `exp` is still in the future

---

## 4. Refresh workflow

### Goal

Continue the **same** single session without re-entering the password.

```text
POST /admin/auth/refresh
{ "refresh_token": "..." }

→ hash must match admin.refresh_token_hash
→ not expired
→ mint new access (+ rotate refresh)
→ OVERWRITE session columns on admin row
→ return new tokens
```

If refresh fails (mismatch / expired / logged out):

```text
401 → client must login again
```

Refresh of a **previous** session after a newer login fails — hashes were overwritten.

---

## 5. Logout workflow

```text
POST /admin/auth/logout
Authorization: Bearer <access_token>   # recommended
```

Server:

```text
UPDATE admin
SET access_token_hash = NULL,
    refresh_token_hash = NULL,
    access_token_expires_at = NULL,
    refresh_token_expires_at = NULL,
    updated_at = NOW()
WHERE lock_key = TRUE
```

Optional: require that the Bearer still matches before clearing (prevents random logout calls). If already overwritten by another login, respond `401` or idempotent `204`.

---

## 6. Password rotation workflow

```text
1. go run ./cmd/admin set-password
2. CLI updates password_hash and clears session columns
3. Any existing access/refresh tokens fail immediately
4. POST /admin/auth/login with the new password
```

No config edit. No server restart required for the password itself (DB already updated). Restart only if you also rotated JWT signing secrets in config.

---

## 7. CRUD workflows (content)

Unchanged pattern once authenticated:

```text
1. Login → access_token
2. Bearer token on /admin/<resource>
3. Middleware confirms current session
4. Repository talks to 000002 tables
5. Honor CHECK / UNIQUE / FK RESTRICT|CASCADE
```

### Profile

```text
GET /admin/profile
PUT /admin/profile          # singleton upsert via profile.lock_key
```

### Publish flow (example: projects)

```text
POST   draft (is_published=false)
PUT    update fields / markdown
PUT    /admin/projects/:id/skills   # replace-all junction
PATCH  is_published=true
DELETE project → CASCADE project_skills only
```

### RESTRICT parents

```text
DELETE skill_categories / social_platforms
  → fail while children exist
  → delete or reassign children first
```

### Education order

```text
ORDER BY start_date DESC, id ASC
```

---

## 8. Error mapping

| Situation | HTTP |
|-----------|------|
| Bad / stale / overwritten access token | 401 |
| Bad refresh | 401 |
| Wrong password | 401 |
| Validation / CHECK | 400 |
| UNIQUE conflict | 409 |
| FK RESTRICT | 409 |
| Missing resource | 404 |

---

## 9. Day-in-the-life

```text
Bootstrap (once)
  migrate 000003
  admin set-password
  configure JWT secrets / TTLs

Morning
  POST /admin/auth/login
  (any other device’s old session dies)

Work
  CRUD with Bearer access_token

Access expires
  POST /admin/auth/refresh
  continue

Done
  POST /admin/auth/logout
  discard client tokens
```

---

## 10. Operational checklist

### First bring-up

- [ ] `000003` migrated
- [ ] `admin set-password` completed
- [ ] Token secrets / TTLs in config (not password)
- [ ] Login rate limit enabled
- [ ] HTTPS on non-local hosts
- [ ] CORS tightened for admin UI origin

### Password change

- [ ] `admin set-password`
- [ ] Confirm old tokens fail
- [ ] Login with new password

### Lost laptop / suspected leak

- [ ] `admin set-password` (clears session) **or** login from a trusted device (overwrites session)
- [ ] Optionally rotate JWT secrets and restart

---

## 11. Out of scope

- Multi-admin users
- Concurrent sessions
- Password in config
- Email reset / OAuth
- Visitor auth
- Portfolio seed data

---

## 12. Next engineering tasks

```text
docs (revised) ✓
    ↓
000003_admin_auth migration
    ↓
cmd/admin set-password + status
    ↓
auth service (verify, overwrite session)
    ↓
/admin/auth/* + session-aware middleware
    ↓
CRUD under /admin
```
```

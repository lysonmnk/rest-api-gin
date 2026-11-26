# Rest-API-Gin Polish & End-to-End Verification Plan

> **For agentic workers:** Follow `superpowers:test-driven-development` (RED → GREEN → REFACTOR), `ponytail` (minimal diff, delete dead code, zero bloat), and `verification-before-completion`.

**Goal:** Eliminate remaining dead code, fix `Login` token-before-session ordering, fix `books.html` port/error/Enter bugs, add browser registration UI in `login.html`, and cover cookie/session auth + validation edge cases in `main_test.go`.

**Files:**
- Modify: `repository/user_repository.go` (remove unused `FindByID`)
- Modify: `handlers/auth_handler.go` (generate JWT before saving session)
- Modify: `templates/books.html` (port-agnostic `API_BASE`, parse JSON error in edit modal, Enter key on search)
- Modify: `templates/login.html` (minimal register toggle wired to `POST /api/register`)
- Modify: `main_test.go` (cookie/session auth + logout test, edge-case validation tests, template checks)

# Security, Testing & Features Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Follow TDD (Red-Green-Refactor) and Ponytail (smallest complete change).

**Goal:** Fix Stored XSS and hardcoded public secrets (Opsi A), isolate tests from the live PostgreSQL database with full CRUD/validation tests (Opsi B), and add user registration (`POST /api/register`) plus book search (`GET /api/books?q=`) (Opsi C).

**Architecture:** Extract `setupRouterWithRepos` in `main.go` so `main_test.go` exercises real Gin routing, session/JWT middleware, Bcrypt, and handlers against in-memory repositories without truncating `gin_books_db`. Keep all changes minimal and idiomatic to the existing layered structure.

**Tech Stack:** Go 1.27, Gin v1.12, GORM/PostgreSQL, `golang.org/x/crypto/bcrypt`, `golang-jwt/jwt/v5`, vanilla JS in `templates/books.html`.

---

### Task 1 (Opsi A): Remove hardcoded public fallback secrets & fix Stored XSS
- **Files:** `config/database.go`, `main.go`, `middleware/auth_middleware.go`, `handlers/auth_handler.go`, `templates/books.html`, `main_test.go`
- **Tests:** Verify random fallback secret differs from old hardcoded strings and `templates/books.html` builds table cells with `textContent` and `addEventListener` instead of `tr.innerHTML` / inline `onclick`.

### Task 2 (Opsi B): Isolate `main_test.go` & add `stock`/`publish_year` validation + missing CRUD/`/api/me` tests
- **Files:** `main.go`, `main_test.go`, `handlers/book_handler.go`
- **Tests (TDD Red → Green):**
  - `TestCreateBookNegativeStock` (`400`)
  - `TestCreateBookInvalidPublishYear` (`400`)
  - `TestGetBookByIDSuccessAndNotFound` (`200`, `404`)
  - `TestUpdateBookSuccessAndNotFound` (`200`, `404`)
  - `TestDeleteBookSuccessAndNotFound` (`204`, `404`)
  - `TestMeWithAuth` (`200`)

### Task 3 (Opsi C): Add `POST /api/register` & `GET /api/books?q=` search
- **Files:** `models/user.go`, `handlers/auth_handler.go`, `repository/book_repository.go`, `handlers/book_handler.go`, `main.go`, `main_test.go`
- **Tests (TDD Red → Green):**
  - `TestRegisterSuccess`, `TestRegisterDuplicateUsername`, `TestRegisterWeakOrEmptyInput`
  - `TestGetBooksSearchQuery`

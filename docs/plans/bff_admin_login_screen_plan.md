---
doc_type: plan
id: BFF-ADMIN-LOGIN
title: "BFF admin login screen plan"
status: in-progress
phase: post-ADR-029
week: 18
tags: [plan, bff, admin, auth, login, session, security]
fr_refs: [FR-230]
uc_refs: [UC-A2]
blocked_by: []
blocks: []
files_affected:
  - bff/src/routes/admin.ts
  - bff/src/routes/adminLayout.ts
  - bff/src/routes/adminAuth.ts
  - bff/src/routes/adminWorkflows.ts
  - bff/tests/adminWorkflows.test.ts
  - bff/scripts/admin-screenshots/shooter.ts
  - bff/scripts/admin-screenshots/catalog.ts
created: 2026-05-05
completed:
---

# BFF admin login screen plan

## Objective

Replace the current bearer-token-only admin entry with a normal browser login
flow for `/bff/admin`. The target outcome is an operator-facing admin surface
that behaves like a standard web application: email/password login, authenticated
server-side session, protected routes, explicit logout, and no token handling in
the visible UI.

## Why this exists

The current admin shell is not usable as a normal operator surface.

- The page exposes a raw `Bearer token` input instead of a login form.
- The token is stored in `localStorage`, which is not the desired security model
  for this surface.
- The current script only injects the bearer token into HTMX-configured requests.
- Most admin navigation is standard browser navigation, so the saved token does
  not authenticate page loads.
- The resulting operator experience is misleading: entering a token appears to
  do nothing, even when the token is valid.

This plan closes that gap by moving authentication responsibility into the BFF
session layer.

## Decision

Use cookie-backed BFF session authentication.

Do not ship a cosmetic login screen that still writes bearer tokens to
`localStorage`. That would preserve the current failure mode and only hide it
behind different wording.

The BFF must:

- accept operator credentials through a normal login form
- authenticate against the existing Go auth endpoint
- store the resulting auth state in a BFF-managed session
- protect all `/bff/admin/*` routes through that session
- remove token handling from the visible admin chrome

## Constraints

- No new authentication product is introduced. Reuse the existing Go auth flow.
- The BFF remains a thin proxy. It may own session state and route protection,
  but it must not introduce business logic or direct database access.
- The admin surface must behave consistently across all admin routes, not only
  workflows.
- The operator UX should stay dense and operational, aligned with the current
  admin and BFF shell style.

## Target user flow

### Login

1. Operator opens `/bff/admin`.
2. If unauthenticated, the BFF redirects to `/bff/admin/login`.
3. The login screen renders a standard form with:
   - `email`
   - `password`
   - primary action `Sign in`
4. `POST /bff/admin/login` relays credentials to the existing Go auth endpoint.
5. On success, the BFF stores the authenticated state in an HTTP-only session
   cookie and redirects the operator to the intended admin destination.

### Authenticated navigation

1. Operator navigates through `/bff/admin/*` routes normally.
2. Protected routes read auth state from the BFF session.
3. The BFF forwards the stored auth token to upstream Go admin requests.
4. The admin shell renders without any bearer-token field.

### Logout

1. Operator clicks `Sign out`.
2. `POST /bff/admin/logout` clears the BFF session.
3. The operator is redirected to `/bff/admin/login`.

## Functional scope

### In scope

- `GET /bff/admin/login` login screen
- `POST /bff/admin/login` credential submission
- BFF-managed session auth for `/bff/admin/*`
- removal of the visible bearer-token field from the admin shell
- `POST /bff/admin/logout`
- redirect-to-login behavior for unauthenticated access
- operator-visible error states for invalid credentials and expired sessions
- focused BFF/Jest coverage for login, logout, redirects, and protected routes

### Out of scope

- SSO
- MFA
- password reset
- user-management UI
- workspace switching UI
- broad auth refactors outside the admin surface

## Security requirements

- Session cookie must be `HttpOnly`.
- Session cookie must be at least `SameSite=Lax`.
- Session cookie must be `Secure` outside local development.
- Raw bearer tokens must not be rendered into HTML.
- Raw bearer tokens must not be persisted in `localStorage` or `sessionStorage`.
- Logout must clear the session state deterministically.
- Upstream unauthorized responses must clear or invalidate stale admin session
  state instead of looping indefinitely.

## Error handling requirements

The login and session model must render explicit operator states for:

- invalid credentials
- expired or revoked upstream session/token
- upstream auth service unavailable
- unauthorized access to a protected admin page

These states must render as normal admin-safe HTML, not raw JSON or transport
errors.

## Proposed implementation slices

### `BAL-01` - Add login route and form

- Effort/reasoning: Medium - introduces a new public admin entry route and
  operator-facing credential UX
- Recommended model: `gpt-5.4`
- Add `GET /bff/admin/login`.
- Add `POST /bff/admin/login`.
- Reuse the existing Go auth endpoint for credential verification.
- Render inline login errors for invalid credentials.

Acceptance:

- unauthenticated operators can reach a standard login page
- valid credentials redirect into the admin surface
- invalid credentials re-render the login form with an explicit error

### `BAL-02` - Replace bearer relay with session-backed admin auth

- Effort/reasoning: Medium - changes the auth transport model for every admin
  route and must remain thin and consistent
- Recommended model: `gpt-5.4`
- Replace or extend `adminBearerRelay` with session-backed auth middleware.
- Store upstream auth token in a server-managed session payload.
- Protect `/bff/admin/*` routes through session presence.
- Remove `Bearer token` input and `localStorage` relay from the admin shell.

Acceptance:

- authenticated admin routes work through normal browser navigation
- the admin shell no longer exposes raw token handling
- stale or missing sessions redirect to `/bff/admin/login`

### `BAL-03` - Add logout and session-expiry handling

- Effort/reasoning: Low - small route surface, but necessary for a complete auth
  loop and clean operator behavior
- Recommended model: `gpt-5.4`
- Add `POST /bff/admin/logout`.
- Clear the session cookie and session payload.
- On upstream `401`, invalidate the session and redirect to login.

Acceptance:

- logout always returns the operator to the login page
- expired upstream auth does not strand the operator in a broken admin state

### `BAL-04` - Add focused test coverage

- Effort/reasoning: Medium - route protection touches multiple flows and needs
  explicit regression coverage
- Recommended model: `gpt-5.4`
- Add tests for login page render.
- Add tests for successful login redirect.
- Add tests for failed login render.
- Add tests for redirecting protected routes when unauthenticated.
- Add tests for logout clearing the session.

Acceptance:

- the admin auth flow is covered at the BFF route level
- route protection behavior is stable under success and failure cases

### `BAL-05` - Update screenshot suite for session-backed admin auth

- Effort/reasoning: Low - targeted replacement of the auth mechanism in one file; no new routes
- Recommended model: `claude-sonnet-4-6`
- Replace bearer-header injection in `shooter.ts` with a Puppeteer login POST.
- Let the browser context carry the session cookie naturally across navigations.
- Add `00_login` catalog entry to capture the unauthenticated login screen.
- Remove all `setExtraHTTPHeaders` and `localStorage` bearer relay code.

Acceptance:

- `npm run admin-screenshots` exits 0 with 14 PNGs + report
- screenshots 01–13 show the authenticated admin shell, not the login redirect
- no bearer token injection remains in the shooter

## Verification plan

Minimum local verification for implementation of this plan:

- `cd bff && npm run lint`
- `cd bff && npm run build`
- `cd bff && npm test`

If implementation touches shared auth or mobile-adjacent paths, expand the QA
gate based on the repo pre-push rules. Do not push before the relevant local
gate passes.

## Handoff notes

This plan is implementation-ready.

The first non-negotiable architectural move is to stop treating the admin
surface as a token relay demo and start treating it as a normal web surface.
If a future implementation keeps `localStorage` token persistence as the
primary auth mechanism, it should be considered a deviation from this plan, not
an equivalent implementation.

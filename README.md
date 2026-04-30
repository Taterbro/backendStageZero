# Intelligence Query Engine

A simple Go REST API that supports natural language processing for querying a database efficiently, with **OAuth-based authentication, role-based access control, and multi-client support (CLI + Web Portal).**

---

This project is my submission for the HNG internship stage 2 backend track.  
https://airtable.com/appZPpwy4dtvVBWU4/shrMH9P1zv4TPhvns?C3OrT=recCRAOnUwTulDtq6

---

## 📦 Tech Stack

- Go (`net/http`)
- UUID for IDs
- errgroup (concurrency handling)
- JWT (access + refresh tokens)
- HTTP-only cookies (web auth)
- GitHub OAuth (authentication provider)
- External APIs:
  - https://api.agify.io
  - https://api.genderize.io
  - https://api.nationalize.io

---

# 🔐 Authentication System

The system supports **GitHub OAuth login** and issues **JWT-based session tokens**.

## 🔁 Authentication Flow

### 1. Login Initiation

- User (CLI or Web) starts login request
- Backend redirects user to GitHub OAuth page

### 2. GitHub Callback

- GitHub redirects back to backend callback endpoint
- Backend exchanges authorization code for GitHub access token
- Backend fetches user profile from GitHub

### 3. Session Creation

Backend then:

- Generates internal **access token (JWT)**
- Generates **refresh token**
- Stores refresh token reference in database
- Associates tokens with user identity + role

### 4. Token Delivery

#### Web Client

- Access token is stored in **HTTP-only secure cookies**
- Prevents JavaScript access (XSS protection)

#### CLI Client

- Tokens are returned in JSON payload
- CLI stores them locally (e.g. `~/.mycli/credentials.json`)

---

## 🔄 Token Refresh Flow

- Access tokens are short-lived
- CLI/Web uses refresh token to request new access token
- Backend validates refresh token against stored record
- New access token is issued without requiring re-login

---

# 🧠 Role-Based Access Control (RBAC)

The system enforces roles at the API level.

## Roles

- `user` → default role
- `admin` → elevated privileges (if enabled in system)

## Enforcement Rules

- Role is embedded inside JWT claims
- Middleware extracts and validates role on every request

### Access Rules Example:

- `/query` → accessible to all authenticated users
- `/admin/*` → restricted to admin role only
- Invalid role → request rejected with `403 Forbidden`

## Security Model

- No role is trusted from client input
- Role is derived only from:
  - JWT claims (validated signature)
  - Server-side stored identity

---

# 💻 CLI + 🌐 Web Integration Model

The backend is designed to support **two different clients simultaneously**:

## 1. CLI Client

Used for developers or terminal-based usage.

### Flow:

- User runs CLI command (e.g. `login`)
- CLI initiates OAuth login via backend
- Backend returns authentication state/code (for polling)
- CLI continuously polls backend for login completion
- Once authenticated:
  - CLI receives JWT + refresh token
- CLI stores credentials locally

### Storage:

- `~/.mycli/credentials.json`

### Behavior:

- Stateless after login
- Uses stored tokens for all API requests

---

## 2. Web Portal

Used for browser-based interaction.

### Flow:

- Browser redirects to GitHub OAuth
- Backend sets HTTP-only cookie after login
- Frontend never sees raw tokens
- Session is automatically attached to requests

### Features:

- Dashboard view
- Profile browsing
- Search interface
- Account page

---

## 🔁 Shared Backend Design

Both CLI and Web:

- Use the same OAuth backend
- Use the same JWT issuance system
- Share the same database session store
- Differ only in token delivery method

| Client | Token Storage    | Auth Method             |
| ------ | ---------------- | ----------------------- |
| CLI    | Local file       | Polling + JSON response |
| Web    | HTTP-only cookie | Browser session         |

---

# ▶️ Running the Project Locally

Make sure Go is installed:

```bash
go version
git clone https://github.com/Taterbro/backendStageZero.git
cd backendStageZero
go mod tidy
go run cmd/api/main.go
```

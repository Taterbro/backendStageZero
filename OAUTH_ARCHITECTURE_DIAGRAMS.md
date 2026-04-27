# OAuth 2.0 Architecture Diagrams

## 1. Complete OAuth Flow

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    OAuth 2.0 with PKCE Authorization Code Flow          │
└─────────────────────────────────────────────────────────────────────────┘

CLIENT                          YOUR SERVER                        GITHUB

  │                                │                                  │
  ├─ (1) GET /auth/github ────────►│                                  │
  │                                │                                  │
  │                                ├─ Generate PKCE Challenge        │
  │                                │   (SHA256 hash)                  │
  │                                │                                  │
  │                                ├─ Generate State Token           │
  │                                │   (64-char random)              │
  │                                │                                  │
  │◄─ (2) HTTP 307 Redirect ───────┤◄─ Redirect to GitHub            │
  │   (with code_challenge         │   authorization page            │
  │    & state)                    │                                  │
  │                                │                                  │
  ├─────── User Authorizes ───────►│                                  │
  │                                │                                  │
  │                                │                                  │
  │                                │◄─ (3) GitHub redirects with code │
  │                                │    & state                       │
  │                                │                                  │
  │◄─ (4) HTTP 307 Redirect ───────┼─ /auth/github/callback?code=... │
  │       (redirect to success)    │                                  │
  │                                │                                  │
  │                                ├─ (5) Validate state token       │
  │                                │                                  │
  │                                ├─ (6) POST to GitHub OAuth       │
  │                                │     endpoint with:              │
  │                                │     - code                      │
  │                                │     - client_id                 │
  │                                │     - client_secret             │
  │                                ├───────────────────────────────► │
  │                                │                                  │
  │                                │◄─ (7) Return access_token       │
  │                                │                                  │
  │                                ├─ (8) GET user info with         │
  │                                │     access_token                │
  │                                ├───────────────────────────────► │
  │                                │                                  │
  │                                │◄─ (9) Return GitHub user data   │
  │                                │     (id, login, email, etc)     │
  │                                │                                  │
  │                                ├─ (10) Create/retrieve user     │
  │                                │      in database                │
  │                                │                                  │
  │                                ├─ (11) Generate token pair      │
  │                                │       - access_token (3 min)    │
  │                                │       - refresh_token (5 min)   │
  │                                │                                  │
  │◄─ (12) HTTP 200 OK ───────────┤                                  │
  │     {                          │                                  │
  │       access_token: "...",     │                                  │
  │       refresh_token: "..."     │                                  │
  │     }                          │                                  │
  │                                │                                  │

SUCCESS! User is authenticated with valid access & refresh tokens.
```

## 2. Token Refresh Flow

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    Refresh Token Flow with Rotation                     │
└─────────────────────────────────────────────────────────────────────────┘

CLIENT                          YOUR SERVER

  │                                │
  ├─ (1) POST /auth/refresh ──────►│
  │       {                        │
  │         refresh_token: "T1"    │
  │       }                        │
  │                                │
  │                                ├─ (2) Validate token T1
  │                                │      - Not expired ✓
  │                                │      - Not invalidated ✓
  │                                │      - Exists in DB ✓
  │                                │
  │                                ├─ (3) IMMEDIATELY mark T1 as invalid
  │                                │      (store in invalidated list)
  │                                │
  │                                ├─ (4) Generate new token pair
  │                                │      - T2 (new access_token)
  │                                │      - T3 (new refresh_token)
  │                                │
  │◄─ (5) HTTP 200 OK ────────────┤
  │       {                        │
  │         access_token: "T2",    │
  │         refresh_token: "T3"    │
  │       }                        │
  │                                │
  ├─ (6) Store T2 & T3 locally    │
  ├─ (7) Discard old T1           │
  │                                │

  (Later, attempt to reuse T1...)

  │                                │
  ├─ POST /auth/refresh ──────────►│
  │     { refresh_token: "T1" }   │
  │                                │
  │                                ├─ Check if T1 is invalidated
  │                                │  (YES - it was marked invalid)
  │                                │
  │◄─ HTTP 401 Unauthorized ──────┤
  │       {                        │
  │         error: "Token has      │
  │         been invalidated"      │
  │       }                        │
  │                                │

SECURITY: Old token (T1) cannot be reused. Prevents token replay attacks.
```

## 3. PKCE Challenge Flow

```
┌─────────────────────────────────────────────────────────────────────────┐
│            PKCE (Proof Key for Code Exchange) Mechanism                 │
└─────────────────────────────────────────────────────────────────────────┘

CLIENT                       YOUR SERVER                      GITHUB

  │                              │                               │
  │                              ├─ (1) Generate Verifier
  │                              │   32 random bytes
  │                              │   = "abc123def456ghi789..."
  │                              │
  │                              ├─ (2) Create Challenge
  │                              │   SHA256(Verifier)
  │                              │   Base64 URL encode
  │                              │   = "E9mlyQt..."
  │                              │
  │                              ├─ (3) Store verifier
  │                              │   in session/memory
  │                              │
  │                              ├─ (4) Send challenge
  │◄─ Redirect to GitHub ────────┤   to GitHub
  │   with code_challenge:       │   code_challenge_method: S256
  │   E9mlyQt...                 │
  │                              ├───────────────────────────────►│
  │                              │                                 │
  │                              │                                 ├─ Store challenge
  │                              │                                 │
  │                              │◄─ (5) User authorizes
  │                              │    GitHub returns code
  │                              │◄─ Redirect with code
  │◄─ Redirect callback ─────────┤◄─ to /auth/github/callback
  │
  ├─ (6) Retrieve stored verifier
  │   "abc123def456ghi789..."
  │
  ├─ POST to GitHub token endpoint with:
  │   - code
  │   - code_verifier: abc123def456ghi789...
  │                               ├─────────────────────────────►│
  │                               │                                 │
  │                               │                                 ├─ (7) Verify
  │                               │                                 │  SHA256(verifier)
  │                               │                                 │  == challenge
  │                               │                                 │  ✓ MATCH!
  │                               │                                 │
  │                               │◄─ (8) Return access_token
  │                               │      Only if verifier matches
  │◄─ Receive access token ───────┤◄────────────────────────────
  │
  SUCCESS! Access token obtained.

  SECURITY: Even if someone intercepts the code, they cannot use it
  without the verifier. The verifier was never sent over the network.
```

## 4. State Token (CSRF Protection)

```
┌─────────────────────────────────────────────────────────────────────────┐
│                State Token CSRF Protection Mechanism                    │
└─────────────────────────────────────────────────────────────────────────┘

LEGITIMATE FLOW:

CLIENT                      YOUR SERVER                    GITHUB

  │                              │                           │
  ├─ (1) GET /auth/github ──────►│                           │
  │                              │                           │
  │                              ├─ Generate State
  │                              │ (32 random bytes)
  │                              │ = "a1b2c3d4e5f6..."
  │                              │
  │                              ├─ Store in session:
  │                              │ state → user_id mapping
  │                              │
  │◄─ Redirect to GitHub ────────┼──────────────────────────►│
  │   state=a1b2c3d4e5f6...      │                           │
  │                              │                           │
  │                              │                           ├─ Return code
  │                              │◄──────────────────────────┤ + state
  │◄─ Redirect callback ─────────┤  with code & state        │
  │   code=xyz123
  │   state=a1b2c3d4e5f6...
  │
  ├─ (2) Check: received state matches session state?
  │   Received: a1b2c3d4e5f6...
  │   Session:  a1b2c3d4e5f6...
  │   ✓ MATCH!
  │
  ├─ Process login successfully
  │


ATTACK FLOW (CSRF):

ATTACKER'S SITE              YOUR SERVER                  VICTIM

  │                              │                           │
  │ (1) Sends fake link:         │                           │
  │ /auth/github/callback        │                           │
  │ code=attacker_code           │                           │
  │ state=fake_state             │                           │
  │                    ───────────────────────────────────►  │
  │                    (victim clicks link)                  │
  │                              │                           │
  │                              │ (2) Check state:
  │                              │ Received: fake_state
  │                              │ Session:  legitimate_state
  │                              │ ✗ MISMATCH!
  │                              │
  │                              ├─ Reject request
  │                              │ Return error
  │                              │
  │ ✓ ATTACK PREVENTED!          │                           │

SECURITY: Attacker cannot forge a valid state token.
Each login gets a unique, unpredictable state.
```

## 5. Token Invalidation Mechanism

```
┌─────────────────────────────────────────────────────────────────────────┐
│              Token Invalidation for Replay Prevention                   │
└─────────────────────────────────────────────────────────────────────────┘

SERVER MEMORY:

┌─────────────────────────────────────┐
│   invalidatedTokens (Map)           │
│                                     │
│   refresh_token_123 : true          │
│   refresh_token_456 : true          │
│   refresh_token_789 : true          │
│                                     │
│   (Tokens marked as used/invalid)   │
└─────────────────────────────────────┘

FLOW:

TIME    USER ACTION              SERVER ACTION

00:00   Has valid refresh token   Token State: valid
        T1

00:10   POST /auth/refresh ──────►✓ Validate T1 (found, not invalid)
        with token T1            ✓ Mark T1 as invalid immediately ⚠️
                                 ✓ Generate new pair (T2, T3)
                                 ◄─ Return T2, T3

00:15   Attempts to reuse T1 ────►✗ Check invalidation list
        POST /auth/refresh        ✗ T1 found in invalidated tokens
        with old token T1         ✗ Return 401 Unauthorized

                                 ◄─ {"error": "token invalidated"}

00:20   Uses valid T3 ───────────►✓ Validate T3 (found, not invalid)
        POST /auth/refresh        ✓ Mark T3 as invalid ⚠️
        with token T3            ✓ Generate new pair (T4, T5)
                                 ◄─ Return T4, T5

RESULT: Each token can only be used ONCE. Prevents:
  - Token replay attacks
  - Session hijacking via stolen tokens
  - Concurrent token usage
```

## 6. Error Handling Flow

```
┌─────────────────────────────────────────────────────────────────────────┐
│                     Error Handling & Recovery                           │
└─────────────────────────────────────────────────────────────────────────┘

OAUTH CALLBACK ERRORS:

Request: GET /auth/github/callback?error=access_denied

                        ┌─────────────────────┐
                        │  Error Handler      │
                        └─────────────────────┘
                                  │
                    ┌─────────────┼─────────────┐
                    │             │             │
            ┌───────▼────────┐   │   ┌─────────▼────────┐
            │ access_denied  │   │   │ access_timeout   │
            └────────────────┘   │   └──────────────────┘
                                 │
                        ┌────────▼─────────┐
                        │ state_mismatch   │
                        └──────────────────┘

Each returns:
{
  "status": "error",
  "message": "User denied authorization"
}
With appropriate HTTP 400/401/502 status code


TOKEN ERRORS:

Request: POST /auth/refresh
Body: {"refresh_token": "invalid_token"}

                    ┌──────────────────────┐
                    │ Token Validator      │
                    └──────────────────────┘
                              │
            ┌─────────────────┼────────────────────┐
            │                 │                    │
    ┌───────▼──────┐  ┌───────▼─────────┐  ┌──────▼──────┐
    │ Not found    │  │ Invalidated     │  │ Expired     │
    └──────────────┘  └─────────────────┘  └─────────────┘

Each returns appropriate 401 Unauthorized:
{
  "status": "error",
  "message": "Refresh token has been invalidated"
}


GITHUB API ERRORS:

Request: github.com/login/oauth/access_token
Response: {"error": "invalid_request"}

                    ┌────────────────────┐
                    │ GitHub Error       │
                    └────────────────────┘
                              │
                    HTTP 502 Bad Gateway
                    {
                      "status": "error",
                      "message": "Failed to exchange authorization code"
                    }
```

## 7. Data Flow Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    Complete Data Flow Architecture                      │
└─────────────────────────────────────────────────────────────────────────┘

                           CLIENT

                              │
                ┌─────────────┼─────────────┐
                │             │             │
        /auth/github    /auth/github/      /auth/refresh
        (initiate)      callback (recv)    /logout


                           HANDLERS
                     (internal/handler)

                    handler/auth.go
                              │
            ┌─────────────────┼─────────────────┐
            │                 │                 │
      GitHubOAuth      CallbackOAuth      RefreshToken
      Handler          Handler            Handler


                          SERVICES
                   (internal/service)

         ┌─────────────────────────────────────┐
         │       oauth.go                      │
         │  - GetGitHubAuthURL()               │
         │  - ExchangeCodeForToken()           │
         │  - GetGitHubUser()                  │
         └─────────────────────────────────────┘
                          │
         ┌─────────────────────────────────────┐
         │       token.go                      │
         │  - GeneratePKCEChallenge()          │
         │  - GenerateStateToken()             │
         │  - GenerateTokenPair()              │
         │  - ValidateRefreshToken()           │
         └─────────────────────────────────────┘


                    EXTERNAL SERVICES

         ┌──────────────────────────────────────┐
         │     GitHub OAuth Endpoints           │
         │                                      │
         │  /login/oauth/authorize              │
         │  /login/oauth/access_token           │
         │  /user                               │
         └──────────────────────────────────────┘


                    DATA STORES

         ┌──────────────────────────────────────┐
         │  In-Memory (Development)             │
         │  invalidatedTokens map               │
         └──────────────────────────────────────┘

         ┌──────────────────────────────────────┐
         │  Database (Production)               │
         │  - users table                       │
         │  - refresh_tokens table              │
         │  - sessions table                    │
         └──────────────────────────────────────┘
```

---

These diagrams provide visual understanding of the OAuth 2.0 flow with PKCE,
token rotation, CSRF protection, and error handling mechanisms.

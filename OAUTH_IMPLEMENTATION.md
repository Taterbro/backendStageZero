# GitHub OAuth 2.0 with PKCE Implementation Guide

## Overview

This implementation provides a secure OAuth 2.0 flow with GitHub as the identity provider, including PKCE (Proof Key for Code Exchange) support and automatic token invalidation for refresh tokens.

## Setup

### 1. Environment Variables

Add the following to your `.env` file:

```env
GITHUB_CLIENT_ID=your_github_client_id
GITHUB_CLIENT_SECRET=your_github_client_secret
GITHUB_REDIRECT_URI=http://localhost:8080/auth/github/callback
```

### 2. GitHub Application Setup

1. Go to GitHub Settings → Developer settings → OAuth Apps
2. Create a new OAuth App with:
   - **Application name**: Your App Name
   - **Homepage URL**: `http://localhost:8080`
   - **Authorization callback URL**: `http://localhost:8080/auth/github/callback`
3. Copy the Client ID and Client Secret

## Endpoints

### 1. Initiate OAuth Flow

**GET** `/auth/github`

Redirects user to GitHub for authentication.

**Response**: Redirect to GitHub authorization page

**Example**:

```bash
curl -L http://localhost:8080/auth/github
```

---

### 2. OAuth Callback Handler

**GET** `/auth/github/callback`

GitHub redirects user back here after authentication.

**Parameters**:

- `code` (query): Authorization code from GitHub
- `state` (query): State token for CSRF protection

**Response**:

```json
{
  "status": "success",
  "access_token": "access_github_12345_1234567890",
  "refresh_token": "refresh_github_12345_1234567890"
}
```

**Error Response**:

```json
{
  "status": "error",
  "message": "OAuth error: access_denied"
}
```

---

### 3. Refresh Token

**POST** `/auth/refresh`

Exchanges a refresh token for a new access and refresh token pair.

**Request**:

```json
{
  "refresh_token": "refresh_github_12345_1234567890"
}
```

**Response**:

```json
{
  "status": "success",
  "access_token": "access_github_12345_1234567890_new",
  "refresh_token": "refresh_github_12345_1234567890_new"
}
```

**Error Response**:

```json
{
  "status": "error",
  "message": "Refresh token has been invalidated"
}
```

**Important**:

- The old refresh token is immediately invalidated
- Each refresh generates a NEW refresh token
- Refresh tokens are valid for 5 minutes
- Access tokens are valid for 3 minutes

---

### 4. Logout

**POST** `/auth/logout`

Invalidates the refresh token server-side.

**Request**:

```json
{
  "refresh_token": "refresh_github_12345_1234567890"
}
```

**Response**:

```json
{
  "status": "success"
}
```

**Error Response**:

```json
{
  "status": "error",
  "message": "Refresh token is required"
}
```

---

## Security Features

### 1. PKCE (Proof Key for Code Exchange)

- Generates cryptographically secure random verifier
- Creates SHA256 challenge from verifier
- Prevents authorization code interception attacks
- Especially important for mobile and SPA applications

### 2. State Token

- CSRF protection mechanism
- State token must match between authorization and callback
- Random 64-character hex string generated per request

### 3. Token Invalidation

- Old refresh tokens are immediately invalidated when new ones are issued
- Prevents token reuse attacks
- Invalidated tokens are stored in memory (use database in production)

### 4. Token Expiration

- **Access Token**: 3 minutes (short-lived for immediate operations)
- **Refresh Token**: 5 minutes (slightly longer to allow refresh before access token expires)

### 5. HTTPS Requirement

- OAuth should only be used over HTTPS in production
- Prevents credential interception

---

## Complete OAuth Flow Diagram

```
1. Client visits /auth/github
   ↓
2. Server generates PKCE challenge and state token
   ↓
3. Server redirects to GitHub authorization URL
   ↓
4. User authorizes application on GitHub
   ↓
5. GitHub redirects to /auth/github/callback with code
   ↓
6. Server exchanges code for GitHub access token
   ↓
7. Server fetches user info from GitHub
   ↓
8. Server creates/retrieves user in database
   ↓
9. Server generates JWT-like access and refresh tokens
   ↓
10. Server returns tokens to client
   ↓
11. Client stores tokens securely
```

---

## Implementation Details

### Token Generation

Currently using simplified token format:

```
access_<userID>_<timestamp>
refresh_<userID>_<timestamp>
```

**For Production**: Implement proper JWT (JSON Web Tokens) using `github.com/golang-jwt/jwt`:

```go
import "github.com/golang-jwt/jwt/v5"

token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
    "sub": userID,
    "exp": time.Now().Add(3 * time.Minute).Unix(),
})
```

### Refresh Token Rotation

The implementation follows the best practice of **refresh token rotation**:

1. Client sends refresh token to `/auth/refresh`
2. Server validates the token
3. Server **immediately invalidates** the old token
4. Server generates a new token pair
5. Server returns new tokens

This prevents token replay attacks - if a refresh token is compromised, it becomes useless after one use.

### State Management

**Current Implementation** (In-Memory):

```go
var invalidatedTokens = make(map[string]bool)
```

**Production Recommendations**:

- Use Redis for distributed state
- Use database with TTL for persistent storage
- Consider session store like `sessions` package

---

## Testing the Flow

### 1. Start OAuth Flow

```bash
curl -L "http://localhost:8080/auth/github"
# This redirects to GitHub login
```

### 2. After Authorization (GitHub redirects to callback)

You'll receive tokens in the response.

### 3. Refresh Token

```bash
curl -X POST "http://localhost:8080/auth/refresh" \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"your_refresh_token_here"}'
```

### 4. Logout

```bash
curl -X POST "http://localhost:8080/auth/logout" \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"your_refresh_token_here"}'
```

---

## Error Handling

### Common Errors

| Error                                   | Cause                      | Solution                                    |
| --------------------------------------- | -------------------------- | ------------------------------------------- |
| `Missing authorization code or state`   | Invalid OAuth callback     | Ensure GitHub redirects with code and state |
| `Refresh token has been invalidated`    | Token already used         | Use the new refresh token from response     |
| `Failed to fetch user information`      | GitHub API rate limit      | Implement exponential backoff               |
| `Failed to exchange authorization code` | Invalid client credentials | Verify GITHUB_CLIENT_ID and CLIENT_SECRET   |

---

## Database Schema (Recommended for Production)

```sql
CREATE TABLE users (
    id VARCHAR(255) PRIMARY KEY,
    github_id INT UNIQUE,
    username VARCHAR(255),
    email VARCHAR(255),
    avatar_url VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE refresh_tokens (
    id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    token VARCHAR(255) UNIQUE NOT NULL,
    is_invalidated BOOLEAN DEFAULT FALSE,
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
```

---

## Next Steps for Production

1. **Implement JWT**: Replace simple token format with proper JWT signing/verification
2. **Database Integration**: Store refresh tokens and user sessions in database
3. **Rate Limiting**: Apply to OAuth endpoints to prevent abuse
4. **HTTPS**: Use TLS certificates for secure communication
5. **Logging**: Implement comprehensive audit logging for security events
6. **Token Encryption**: Encrypt refresh tokens at rest in database
7. **CORS Configuration**: Fine-tune CORS policies per environment
8. **Scope Management**: Request only necessary GitHub scopes
9. **Session Management**: Implement secure session handling
10. **Monitoring**: Set up alerts for suspicious authentication patterns

---

## References

- [OAuth 2.0 Specification](https://tools.ietf.org/html/rfc6749)
- [PKCE (RFC 7636)](https://tools.ietf.org/html/rfc7636)
- [GitHub OAuth Documentation](https://docs.github.com/en/developers/apps/building-oauth-apps)
- [JWT Best Practices](https://tools.ietf.org/html/rfc8725)

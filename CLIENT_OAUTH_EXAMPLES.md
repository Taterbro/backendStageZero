# Client-Side OAuth Implementation Examples

## JavaScript/React Example

### 1. Initiate OAuth Flow

```javascript
// Start the OAuth flow
function startGitHubAuth() {
  window.location.href = "http://localhost:8080/auth/github";
}
```

### 2. Handle Callback

The callback will return tokens directly in the response. You'll need a callback component:

```javascript
import React, { useEffect } from "react";
import { useSearchParams, useNavigate } from "react-router-dom";

export function OAuthCallback() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();

  useEffect(() => {
    // The backend will return tokens in the response
    // You need to parse them from the redirect or use a separate callback endpoint
    const code = searchParams.get("code");
    if (code) {
      // Tokens are returned from /auth/github/callback
      console.log("Authorization successful");
      navigate("/dashboard");
    }
  }, []);

  return <div>Processing authentication...</div>;
}
```

### 3. Store Tokens Securely

```javascript
// Store tokens (use secure storage in production)
function storeTokens(accessToken, refreshToken) {
  // For web: use secure HttpOnly cookies (server should set this)
  // For SPA: use localStorage with XSS protection
  localStorage.setItem("access_token", accessToken);
  localStorage.setItem("refresh_token", refreshToken);
}

// Retrieve tokens
function getAccessToken() {
  return localStorage.getItem("access_token");
}

function getRefreshToken() {
  return localStorage.getItem("refresh_token");
}
```

### 4. Refresh Token

```javascript
async function refreshAccessToken() {
  const refreshToken = getRefreshToken();

  try {
    const response = await fetch("http://localhost:8080/auth/refresh", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        refresh_token: refreshToken,
      }),
    });

    if (!response.ok) {
      throw new Error("Token refresh failed");
    }

    const data = await response.json();

    // Store new tokens (old ones are invalidated by server)
    storeTokens(data.access_token, data.refresh_token);

    return data.access_token;
  } catch (error) {
    console.error("Refresh failed:", error);
    // Redirect to login
    logout();
  }
}
```

### 5. API Requests with Token

```javascript
async function fetchWithAuth(url, options = {}) {
  let accessToken = getAccessToken();

  const headers = {
    "Content-Type": "application/json",
    Authorization: `Bearer ${accessToken}`,
    ...options.headers,
  };

  let response = await fetch(url, {
    ...options,
    headers,
  });

  // If 401, try refreshing token
  if (response.status === 401) {
    try {
      accessToken = await refreshAccessToken();
      headers["Authorization"] = `Bearer ${accessToken}`;
      response = await fetch(url, {
        ...options,
        headers,
      });
    } catch (error) {
      logout();
      throw error;
    }
  }

  return response;
}
```

### 6. Logout

```javascript
async function logout() {
  const refreshToken = getRefreshToken();

  try {
    await fetch("http://localhost:8080/auth/logout", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        refresh_token: refreshToken,
      }),
    });
  } finally {
    // Clear local tokens
    localStorage.removeItem("access_token");
    localStorage.removeItem("refresh_token");
    window.location.href = "/login";
  }
}
```

### 7. React Hook for Auth

```javascript
import { useCallback, useState, useEffect } from "react";

function useAuth() {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    // Check if tokens exist on mount
    const accessToken = localStorage.getItem("access_token");
    setIsAuthenticated(!!accessToken);
    setIsLoading(false);
  }, []);

  const login = useCallback(() => {
    window.location.href = "http://localhost:8080/auth/github";
  }, []);

  const logout = useCallback(async () => {
    const refreshToken = localStorage.getItem("refresh_token");
    await fetch("http://localhost:8080/auth/logout", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: refreshToken }),
    }).finally(() => {
      localStorage.removeItem("access_token");
      localStorage.removeItem("refresh_token");
      setIsAuthenticated(false);
    });
  }, []);

  const getAccessToken = useCallback(() => {
    return localStorage.getItem("access_token");
  }, []);

  return {
    isAuthenticated,
    isLoading,
    login,
    logout,
    getAccessToken,
  };
}

export default useAuth;
```

---

## cURL Examples

### 1. Initiate OAuth

```bash
curl -L "http://localhost:8080/auth/github"
```

### 2. Refresh Token

```bash
curl -X POST "http://localhost:8080/auth/refresh" \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "refresh_github_user123_1234567890"
  }'
```

Response:

```json
{
  "status": "success",
  "access_token": "access_github_user123_new",
  "refresh_token": "refresh_github_user123_new"
}
```

### 3. Logout

```bash
curl -X POST "http://localhost:8080/auth/logout" \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "refresh_github_user123_1234567890"
  }'
```

---

## Python Example

```python
import requests
import json
from datetime import datetime, timedelta

class GitHubAuth:
    def __init__(self, base_url='http://localhost:8080'):
        self.base_url = base_url
        self.access_token = None
        self.refresh_token = None
        self.token_expiry = None

    def initiate_oauth(self):
        """Start OAuth flow"""
        auth_url = f"{self.base_url}/auth/github"
        return auth_url

    def refresh_token_flow(self):
        """Refresh access token"""
        response = requests.post(
            f"{self.base_url}/auth/refresh",
            json={"refresh_token": self.refresh_token}
        )

        if response.status_code == 200:
            data = response.json()
            self.access_token = data['access_token']
            self.refresh_token = data['refresh_token']
            self.token_expiry = datetime.now() + timedelta(minutes=3)
            return True

        return False

    def logout(self):
        """Logout and invalidate refresh token"""
        response = requests.post(
            f"{self.base_url}/auth/logout",
            json={"refresh_token": self.refresh_token}
        )

        if response.status_code == 200:
            self.access_token = None
            self.refresh_token = None
            self.token_expiry = None
            return True

        return False

    def api_request(self, method, endpoint, **kwargs):
        """Make authenticated API request"""
        if self.is_token_expired():
            self.refresh_token_flow()

        headers = kwargs.get('headers', {})
        headers['Authorization'] = f"Bearer {self.access_token}"
        kwargs['headers'] = headers

        url = f"{self.base_url}{endpoint}"
        return requests.request(method, url, **kwargs)

    def is_token_expired(self):
        """Check if token is expired"""
        if not self.token_expiry:
            return True
        return datetime.now() >= self.token_expiry

# Usage
auth = GitHubAuth()
print(f"Visit: {auth.initiate_oauth()}")
# After OAuth flow completes and you get tokens:
# auth.access_token = "token_here"
# auth.refresh_token = "refresh_token_here"

# Make authenticated request
# response = auth.api_request('GET', '/api/profiles')
```

---

## Security Best Practices for Client

1. **Store tokens securely**

   - Use HttpOnly cookies (server sets this)
   - Never store sensitive tokens in localStorage if possible
   - Use sessionStorage for temporary access tokens

2. **CSRF Protection**

   - Validate state token matches (server does this)
   - Use SameSite cookie flag

3. **Prevent XSS**

   - Use Content Security Policy (CSP)
   - Sanitize user input
   - Avoid storing tokens in JavaScript accessible storage

4. **Token Refresh**

   - Refresh before expiration (refresh at 2.5 minutes)
   - Use exponential backoff on failed refresh
   - Clear tokens on 401 response

5. **HTTPS**

   - Always use HTTPS in production
   - Use secure flag on cookies
   - Implement HSTS header

6. **Session Management**
   - Implement session timeout
   - Clear tokens on logout
   - Handle token expiration gracefully

---

## Common Issues & Solutions

| Issue                 | Solution                                                        |
| --------------------- | --------------------------------------------------------------- |
| Tokens not persisting | Check localStorage/cookies settings, ensure HTTPS in production |
| 401 Unauthorized      | Refresh token and retry, or redirect to login                   |
| CORS errors           | Ensure backend has proper CORS headers set                      |
| State mismatch        | Clear browser cache, ensure /auth/github is called first        |
| Rate limiting         | Add exponential backoff to refresh attempts                     |

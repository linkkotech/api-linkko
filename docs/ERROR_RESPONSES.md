# Error Response Specification

## Overview

All API error responses follow a consistent JSON structure:

```json
{
  "ok": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "fields": {
      "field_name": "field-specific error"
    }
  }
}
```

## HTTP Status Code Usage

### 400 Bad Request
Used for **validation errors** in request parameters, path variables, or body.

**When to use:**
- Invalid format/syntax in path parameters
- Missing required parameters
- Constraint violations (length, pattern, etc.)

### 401 Unauthorized
Used for **authentication failures** only.

**When to use:**
- Missing Authorization header
- Invalid/expired JWT token
- Invalid token signature
- Malformed authentication credentials

### 403 Forbidden
Used for **authorization failures** - user is authenticated but lacks permission.

**When to use:**
- Workspace access denied (IDOR protection)
- Insufficient scope/permissions
- Resource belongs to different workspace

### 500 Internal Server Error
Used for **unexpected server errors**.

**When to use:**
- Database connection failures
- Unexpected exceptions
- Service unavailable

---

## Error Codes Reference

### 400 Bad Request Codes

#### `INVALID_WORKSPACE_ID`
WorkspaceID contains invalid characters or exceeds length limit.

**Example Request:**
```http
GET /api/workspaces/invalid@workspace/contacts
Authorization: Bearer eyJhbGc...
```

**Response (400):**
```json
{
  "ok": false,
  "error": {
    "code": "INVALID_WORKSPACE_ID",
    "message": "workspaceId must contain only alphanumeric characters, hyphens, and underscores (max 64 chars)"
  }
}
```

---

#### `MISSING_PARAMETER`
Required path parameter is missing or empty.

**Example Request:**
```http
GET /api/workspaces//contacts
Authorization: Bearer eyJhbGc...
```

**Response (400):**
```json
{
  "ok": false,
  "error": {
    "code": "MISSING_PARAMETER",
    "message": "workspaceId is required in path"
  }
}
```

---

#### `INVALID_PARAMETER`
S2S authentication headers are invalid or missing.

**Example Request:**
```http
POST /api/workspaces/my-workspace/tasks
Authorization: Bearer s2s_token_here
X-Workspace-Id: 
X-Actor-Id: user123
```

**Response (400):**
```json
{
  "ok": false,
  "error": {
    "code": "INVALID_PARAMETER",
    "message": "invalid X-Workspace-Id or X-Actor-Id header"
  }
}
```

---

#### `VALIDATION_ERROR`
Request body contains validation errors (with field-level details).

**Example Request:**
```http
POST /api/workspaces/my-workspace/contacts
Authorization: Bearer eyJhbGc...
Content-Type: application/json

{
  "email": "not-an-email",
  "name": ""
}
```

**Response (400):**
```json
{
  "ok": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "validation failed",
    "fields": {
      "email": "invalid email format",
      "name": "name is required"
    }
  }
}
```

---

### 401 Unauthorized Codes

#### `MISSING_AUTHORIZATION`
Authorization header is not present in the request.

**Example Request:**
```http
GET /api/workspaces/my-workspace/contacts
```

**Response (401):**
```json
{
  "ok": false,
  "error": {
    "code": "MISSING_AUTHORIZATION",
    "message": "missing authorization header"
  }
}
```

---

#### `INVALID_SCHEME`
Authorization header uses wrong scheme (expected: Bearer).

**Example Request:**
```http
GET /api/workspaces/my-workspace/contacts
Authorization: Basic dXNlcjpwYXNz
```

**Response (401):**
```json
{
  "ok": false,
  "error": {
    "code": "INVALID_SCHEME",
    "message": "invalid authorization scheme, expected Bearer"
  }
}
```

---

#### `INVALID_TOKEN`
JWT token is malformed or invalid.

**Example Request:**
```http
GET /api/workspaces/my-workspace/contacts
Authorization: Bearer invalid.token.here
```

**Response (401):**
```json
{
  "ok": false,
  "error": {
    "code": "INVALID_TOKEN",
    "message": "invalid token"
  }
}
```

---

#### `TOKEN_EXPIRED`
JWT token has expired.

**Example Request:**
```http
GET /api/workspaces/my-workspace/contacts
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Response (401):**
```json
{
  "ok": false,
  "error": {
    "code": "TOKEN_EXPIRED",
    "message": "token expired"
  }
}
```

---

#### `INVALID_SIGNATURE`
Token signature verification failed (JWT or S2S).

**Example Request:**
```http
POST /api/workspaces/my-workspace/tasks
Authorization: Bearer s2s_tampered_signature
X-Workspace-Id: my-workspace
X-Actor-Id: user123
```

**Response (401):**
```json
{
  "ok": false,
  "error": {
    "code": "INVALID_SIGNATURE",
    "message": "invalid S2S token"
  }
}
```

---

#### `INVALID_ISSUER`
Token issuer does not match expected value.

**Example Request:**
```http
GET /api/workspaces/my-workspace/contacts
Authorization: Bearer eyJhbGc...
```

**Response (401):**
```json
{
  "ok": false,
  "error": {
    "code": "INVALID_ISSUER",
    "message": "invalid token issuer"
  }
}
```

---

#### `INVALID_AUDIENCE`
Token audience does not match expected value.

**Example Request:**
```http
GET /api/workspaces/my-workspace/contacts
Authorization: Bearer eyJhbGc...
```

**Response (401):**
```json
{
  "ok": false,
  "error": {
    "code": "INVALID_AUDIENCE",
    "message": "invalid token audience"
  }
}
```

---

### 403 Forbidden Codes

#### `WORKSPACE_MISMATCH`
Authenticated user/service is trying to access a workspace they don't belong to (IDOR protection).

**Example Request:**
```http
GET /api/workspaces/other-workspace/contacts
Authorization: Bearer eyJhbGc... (token with workspace_id: "my-workspace")
```

**Response (403):**
```json
{
  "ok": false,
  "error": {
    "code": "WORKSPACE_MISMATCH",
    "message": "workspace access denied"
  }
}
```

**S2S Example:**
```http
POST /api/workspaces/other-workspace/tasks
Authorization: Bearer s2s_token
X-Workspace-Id: my-workspace
X-Actor-Id: service-account
```

**Response (403):**
```json
{
  "ok": false,
  "error": {
    "code": "WORKSPACE_MISMATCH",
    "message": "workspace access denied"
  }
}
```

---

#### `FORBIDDEN`
Generic authorization failure (insufficient permissions, scope issues).

**Example Request:**
```http
DELETE /api/workspaces/my-workspace/users/admin-user
Authorization: Bearer eyJhbGc... (token with role: "user")
```

**Response (403):**
```json
{
  "ok": false,
  "error": {
    "code": "FORBIDDEN",
    "message": "insufficient permissions"
  }
}
```

---

#### `INSUFFICIENT_SCOPE`
Token lacks required OAuth scope for the operation.

**Example Request:**
```http
POST /api/workspaces/my-workspace/contacts
Authorization: Bearer eyJhbGc... (token with scope: "read:contacts")
```

**Response (403):**
```json
{
  "ok": false,
  "error": {
    "code": "INSUFFICIENT_SCOPE",
    "message": "required scope: write:contacts"
  }
}
```

---

### 500 Internal Server Error Codes

#### `INTERNAL_ERROR`
Unexpected server-side error.

**Example Request:**
```http
GET /api/workspaces/my-workspace/contacts
Authorization: Bearer eyJhbGc...
```

**Response (500):**
```json
{
  "ok": false,
  "error": {
    "code": "INTERNAL_ERROR",
    "message": "an unexpected error occurred"
  }
}
```

---

## Integration Examples

### cURL Examples

**Authentication Failure (401):**
```bash
curl -X GET https://api.linkko.com/api/workspaces/my-workspace/contacts

# Response:
# HTTP/1.1 401 Unauthorized
# Content-Type: application/json
# {"ok":false,"error":{"code":"MISSING_AUTHORIZATION","message":"missing authorization header"}}
```

**Workspace Mismatch (403):**
```bash
curl -X GET https://api.linkko.com/api/workspaces/other-workspace/contacts \
  -H "Authorization: Bearer eyJhbGc..."

# Response:
# HTTP/1.1 403 Forbidden
# Content-Type: application/json
# {"ok":false,"error":{"code":"WORKSPACE_MISMATCH","message":"workspace access denied"}}
```

**Invalid WorkspaceID (400):**
```bash
curl -X GET https://api.linkko.com/api/workspaces/invalid@workspace!/contacts \
  -H "Authorization: Bearer eyJhbGc..."

# Response:
# HTTP/1.1 400 Bad Request
# Content-Type: application/json
# {"ok":false,"error":{"code":"INVALID_WORKSPACE_ID","message":"workspaceId must contain only alphanumeric characters, hyphens, and underscores (max 64 chars)"}}
```

---

## Client Implementation Guide

### JavaScript/TypeScript

```typescript
interface ErrorResponse {
  ok: false;
  error: {
    code: string;
    message: string;
    fields?: Record<string, string>;
  };
}

async function handleApiRequest(url: string, token: string) {
  const response = await fetch(url, {
    headers: { Authorization: `Bearer ${token}` }
  });
  
  if (!response.ok) {
    const error: ErrorResponse = await response.json();
    
    switch (response.status) {
      case 400:
        console.error('Validation error:', error.error.message);
        if (error.error.fields) {
          console.error('Field errors:', error.error.fields);
        }
        break;
      
      case 401:
        console.error('Authentication failed:', error.error.code);
        // Redirect to login or refresh token
        break;
      
      case 403:
        console.error('Access denied:', error.error.message);
        // Show "no permission" message
        break;
      
      case 500:
        console.error('Server error:', error.error.message);
        // Show generic error message
        break;
    }
    
    throw new Error(error.error.message);
  }
  
  return response.json();
}
```

### Go Client

```go
package client

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type ErrorResponse struct {
	OK    bool        `json:"ok"`
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func HandleAPIRequest(url, token string) error {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode >= 400 {
		var errResp ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return fmt.Errorf("failed to decode error response: %w", err)
		}
		
		switch resp.StatusCode {
		case http.StatusBadRequest:
			return fmt.Errorf("validation error: %s (code: %s)", 
				errResp.Error.Message, errResp.Error.Code)
		case http.StatusUnauthorized:
			return fmt.Errorf("authentication failed: %s", errResp.Error.Code)
		case http.StatusForbidden:
			return fmt.Errorf("access denied: %s", errResp.Error.Message)
		case http.StatusInternalServerError:
			return fmt.Errorf("server error: %s", errResp.Error.Message)
		}
	}
	
	return nil
}
```

---

## Testing Error Responses

### Unit Test Example

```go
func TestWorkspaceMiddleware_ErrorResponses(t *testing.T) {
	tests := []struct {
		name           string
		workspaceID    string
		setupAuth      func(*http.Request)
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "InvalidWorkspaceID",
			workspaceID:    "invalid@workspace",
			setupAuth:      func(r *http.Request) { /* no auth needed */ },
			expectedStatus: 400,
			expectedCode:   "INVALID_WORKSPACE_ID",
		},
		{
			name:           "MissingAuthorization",
			workspaceID:    "valid-workspace",
			setupAuth:      func(r *http.Request) { /* no auth */ },
			expectedStatus: 401,
			expectedCode:   "INVALID_TOKEN",
		},
		{
			name:           "WorkspaceMismatch",
			workspaceID:    "other-workspace",
			setupAuth: func(r *http.Request) {
				ctx := auth.SetAuthContextForTesting(r.Context(), &auth.AuthContext{
					WorkspaceID: "my-workspace",
				})
				*r = *r.WithContext(ctx)
			},
			expectedStatus: 403,
			expectedCode:   "WORKSPACE_MISMATCH",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/workspaces/"+tt.workspaceID+"/contacts", nil)
			tt.setupAuth(req)
			
			rec := httptest.NewRecorder()
			middleware.WorkspaceMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(200)
			})).ServeHTTP(rec, req)
			
			assert.Equal(t, tt.expectedStatus, rec.Code)
			
			var errResp ErrorResponse
			json.Unmarshal(rec.Body.Bytes(), &errResp)
			assert.False(t, errResp.OK)
			assert.Equal(t, tt.expectedCode, errResp.Error.Code)
		})
	}
}
```

---

## Summary

✅ **401 Unauthorized**: Authentication problems (missing/invalid token)
✅ **403 Forbidden**: Authorization problems (authenticated but no permission)
✅ **400 Bad Request**: Validation errors (invalid parameters/body)
✅ **500 Internal Server Error**: Unexpected server errors

All responses follow consistent JSON structure with error codes for client-side handling.

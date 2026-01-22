package auth

import "context"

// SetAuthContextForTesting injects an AuthContext into a context for testing purposes
// This should only be used in tests to simulate authenticated requests
func SetAuthContextForTesting(ctx context.Context, authCtx *AuthContext) context.Context {
	return context.WithValue(ctx, authContextKey, authCtx)
}

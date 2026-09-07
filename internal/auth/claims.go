// Package auth (claims.go) defines the per-request identity used by handlers.
//
// Claims is attached to *http.Request.Context by RequireAuth and read with
// ClaimsFromContext. A zero Claims value (or !ok) means “unauthenticated.”
package auth

import (
	"context"

	"tradielynx/internal/models"
)

type ctxKey int

// ClaimsKey is the context key under which we store auth.Claims.
var claimsKey ctxKey // unexported so only this package can set/read it.

// Attaches c to ctx and returns the derived context.
func WithClaims(ctx context.Context, c models.Claims) context.Context {
	return context.WithValue(ctx, claimsKey, c)
}

// Extracts Claims previously attached by WithClaims.
// If no Claims are present, ok will be false and the returned Claims is zero.
func ClaimsFromContext(ctx context.Context) (models.Claims, bool) {
	c, ok := ctx.Value(claimsKey).(models.Claims)
	return c, ok
}

package auth

import (
	"net/http"
	"tradielynx/internal/models"
)

// Guard adds auth (and optional role checks) to a handler.
//
// roles is variadic (…models.Role): pass zero roles to allow any logged-in user,
// or one/more roles to require the user to match at least one of them.
func Guard(h http.Handler, roles ...models.Role) http.Handler {
	if len(roles) == 0 {
		return RequireAuth(h) // any logged-in user
	}
	return RequireAuth(RequireRole(roles...)(h))
}

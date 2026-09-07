package session

import (
	"net/http"
	"time"

	"github.com/alexedwards/scs/v2"
)

var Manager *scs.SessionManager

// Initlializes session (cookie)
func InitSession() {
	sm := scs.New()
	sm.Lifetime = 24 * time.Hour
	sm.IdleTimeout = 90 * time.Minute
	sm.Cookie.Name = "session_id"
	sm.Cookie.Path = "/"     // make it valid for the whole site
	sm.Cookie.Persist = true // keep cookie after browser restarts (optional)
	sm.Cookie.HttpOnly = true
	sm.Cookie.SameSite = http.SameSiteLaxMode
	sm.Cookie.Secure = false // set true in HTTPS
	Manager = sm
}

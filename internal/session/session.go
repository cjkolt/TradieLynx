package session

import (
	"net/http"
	"time"

	"github.com/alexedwards/scs/v2"
)

var Manager *scs.SessionManager

func InitSession() {
	sm := scs.New()
	sm.Lifetime = 24 * time.Hour
	sm.IdleTimeout = 90 * time.Minute
	sm.Cookie.Name = "session_id"
	sm.Cookie.Path = "/"    
	sm.Cookie.Persist = true
	sm.Cookie.HttpOnly = true
	sm.Cookie.SameSite = http.SameSiteLaxMode
	sm.Cookie.Secure = false
	Manager = sm
}

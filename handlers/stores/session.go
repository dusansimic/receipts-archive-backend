package stores

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
)

// Session is an options struct for the session
type Session struct {
	SessionOptions sessions.Options
	Secret         []byte
}

// NewSessionStore creates new store for the session
func (s Session) NewSessionStore() cookie.Store {
	store := cookie.NewStore(s.Secret)
	store.Options(s.SessionOptions)
	return store
}

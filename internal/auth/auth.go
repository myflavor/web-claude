package auth

import (
	"crypto/subtle"
	"net/http"
	"strings"
	"time"
)

const CookieName = "cm_token"

type Guard struct {
	token        string
	cookieSecure bool
}

func New(token string, cookieSecure bool) *Guard {
	return &Guard{
		token:        token,
		cookieSecure: cookieSecure,
	}
}

func (g *Guard) ValidToken(got string) bool {
	if got == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(g.token)) == 1
}

// ExtractToken looks at Authorization Bearer, cookie, then query ?token= (WS bootstrap).
func (g *Guard) ExtractToken(r *http.Request) string {
	if h := r.Header.Get("Authorization"); h != "" {
		const p = "Bearer "
		if strings.HasPrefix(h, p) {
			return strings.TrimSpace(h[len(p):])
		}
	}
	if c, err := r.Cookie(CookieName); err == nil && c.Value != "" {
		return c.Value
	}
	if t := r.URL.Query().Get("token"); t != "" {
		return t
	}
	return ""
}

func (g *Guard) Authorized(r *http.Request) bool {
	return g.ValidToken(g.ExtractToken(r))
}

func (g *Guard) SetCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   g.cookieSecure,
		// 30 days
		Expires: time.Now().Add(30 * 24 * time.Hour),
	})
}

func (g *Guard) ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   g.cookieSecure,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

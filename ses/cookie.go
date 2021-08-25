package ses

import (
	"net/http"
	"time"
)

// Cookie is a wrapped http cookie that implements token reader and writer.
type Cookie http.Cookie

// DefaultCookie returns a cookie using name and secure and other default parameters.
// The defaults are http only, root path and same site lax mode.
func DefaultCookie(name string, secure bool) *Cookie {
	return &Cookie{
		Name:     name,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
		Path:     "/",
	}
}

func (cc *Cookie) ReadToken(r *http.Request) string {
	if c, err := r.Cookie(cc.Name); err == nil {
		return c.Value
	}
	return ""
}

func (cc *Cookie) WriteToken(w http.ResponseWriter, tok string) {
	c := http.Cookie(*cc)
	c.Value = tok
	if tok == "" {
		c.MaxAge = -1
		c.Expires = time.Unix(1, 0)
	}
	http.SetCookie(w, &c)
}

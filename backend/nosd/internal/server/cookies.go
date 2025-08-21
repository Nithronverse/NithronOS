package server

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"time"

	"nithronos/backend/nosd/internal/config"

	"github.com/gorilla/securecookie"
)

const (
	cookieSession = "nos_session"
	cookieRefresh = "nos_refresh"
	cookieCSRF    = "nos_csrf"
)

// issueSessionCookies sets nos_session (15m) and optionally rotates/sets nos_refresh (7d)
func issueSessionCookies(w http.ResponseWriter, cfg config.Config, uid string, keepRefresh bool) error {
	now := time.Now().UTC()
	// session token
	sess := map[string]any{"uid": uid, "exp": now.Add(15 * time.Minute).Unix()}
	sVal, err := encodeOpaque(cfg, cookieSession, sess)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{Name: cookieSession, Value: sVal, Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, Expires: now.Add(15 * time.Minute)})
	// refresh
	if keepRefresh {
		ref := map[string]any{"uid": uid, "exp": now.Add(7 * 24 * time.Hour).Unix()}
		rVal, err := encodeOpaque(cfg, cookieRefresh, ref)
		if err != nil {
			return err
		}
		http.SetCookie(w, &http.Cookie{Name: cookieRefresh, Value: rVal, Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, Expires: now.Add(7 * 24 * time.Hour)})
	}
	return nil
}

func clearAuthCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: cookieSession, Value: "", Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, MaxAge: -1})
	http.SetCookie(w, &http.Cookie{Name: cookieRefresh, Value: "", Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, MaxAge: -1})
	http.SetCookie(w, &http.Cookie{Name: cookieCSRF, Value: "", Path: "/", Secure: true, SameSite: http.SameSiteLaxMode, MaxAge: -1})
}

// decodeSessionUID validates nos_session and returns uid string
func decodeSessionUID(r *http.Request, cfg config.Config) (string, bool) {
	ck, err := r.Cookie(cookieSession)
	if err != nil {
		return "", false
	}
	var m map[string]any
	if err := decodeOpaque(cfg, cookieSession, ck.Value, &m); err != nil {
		return "", false
	}
	// exp check (support multiple numeric/string types)
	expUnix, ok := asInt64(m["exp"])
	if !ok || time.Now().UTC().Unix() > expUnix {
		return "", false
	}
	if uid, ok := m["uid"].(string); ok && uid != "" {
		return uid, true
	}
	return "", false
}

// decodeRefreshUID validates nos_refresh and returns uid string
func decodeRefreshUID(r *http.Request, cfg config.Config) (string, bool) {
	ck, err := r.Cookie(cookieRefresh)
	if err != nil {
		return "", false
	}
	var m map[string]any
	if err := decodeOpaque(cfg, cookieRefresh, ck.Value, &m); err != nil {
		return "", false
	}
	expUnix, ok := asInt64(m["exp"])
	if !ok || time.Now().UTC().Unix() > expUnix {
		return "", false
	}
	if uid, ok := m["uid"].(string); ok && uid != "" {
		return uid, true
	}
	return "", false
}

func issueCSRFCookie(w http.ResponseWriter) {
	b := securecookie.GenerateRandomKey(32)
	http.SetCookie(w, &http.Cookie{Name: cookieCSRF, Value: encodeBase64(b), Path: "/", Secure: true, SameSite: http.SameSiteLaxMode, Expires: time.Now().Add(24 * time.Hour)})
}

func encodeOpaque(cfg config.Config, name string, payload map[string]any) (string, error) {
	key, _ := json.Marshal([]byte{}) // dummy to satisfy interface
	_ = key
	secret, err := os.ReadFile(cfg.SecretPath)
	if err != nil || len(secret) == 0 {
		secret = cfg.SessionHashKey
	}
	sc := securecookie.New(secret, nil)
	sc.MaxAge(0)
	return sc.Encode(name, payload)
}

func decodeOpaque(cfg config.Config, name string, val string, out *map[string]any) error {
	secret, err := os.ReadFile(cfg.SecretPath)
	if err != nil || len(secret) == 0 {
		secret = cfg.SessionHashKey
	}
	sc := securecookie.New(secret, nil)
	sc.MaxAge(0)
	return sc.Decode(name, val, out)
}

func encodeBase64(b []byte) string {
	// lightweight encoder without pulling extra deps
	v, _ := securecookie.EncodeMulti("b64", b, securecookie.CodecsFromPairs()...)
	return v
}

func asInt64(v any) (int64, bool) {
	switch x := v.(type) {
	case int64:
		return x, true
	case int:
		return int64(x), true
	case float64:
		return int64(x), true
	case string:
		if n, err := strconv.ParseInt(x, 10, 64); err == nil {
			return n, true
		}
	}
	return 0, false
}

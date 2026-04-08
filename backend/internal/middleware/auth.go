package middleware

import (
	"context"
	"net/http"

	"github.com/spaceballone/backend/internal/auth"
	"github.com/spaceballone/backend/internal/models"
	"gorm.io/gorm"
)

type contextKey string

const (
	SessionContextKey contextKey = "session"
	UserContextKey    contextKey = "user"
)

// SessionCookieName is the name of the session cookie.
const SessionCookieName = "spaceballone_session"

// AuthMiddleware validates the session cookie and injects user/session into context.
// It skips auth for /api/health and /api/auth/login.
func AuthMiddleware(db *gorm.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path

			// Skip auth for public endpoints
			if path == "/api/health" || path == "/api/auth/login" {
				next.ServeHTTP(w, r)
				return
			}

			cookie, err := r.Cookie(SessionCookieName)
			if err != nil || cookie.Value == "" {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			session, err := auth.ValidateSession(db, cookie.Value)
			if err != nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			var user models.User
			if err := db.First(&user, "id = ?", session.UserID).Error; err != nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			// If user must change password, block all non-auth endpoints
			if user.MustChangePassword && !isAuthEndpoint(path) {
				http.Error(w, `{"error":"must_change_password"}`, http.StatusForbidden)
				return
			}

			ctx := context.WithValue(r.Context(), SessionContextKey, session)
			ctx = context.WithValue(ctx, UserContextKey, &user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func isAuthEndpoint(path string) bool {
	switch path {
	case "/api/auth/login", "/api/auth/logout", "/api/auth/me", "/api/auth/change-password":
		return true
	}
	return false
}

// GetUser retrieves the authenticated user from the request context.
func GetUser(r *http.Request) *models.User {
	user, _ := r.Context().Value(UserContextKey).(*models.User)
	return user
}

// GetSession retrieves the app session from the request context.
func GetSession(r *http.Request) *models.AppSession {
	session, _ := r.Context().Value(SessionContextKey).(*models.AppSession)
	return session
}

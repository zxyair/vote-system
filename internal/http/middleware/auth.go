package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	HeaderUserID   = "X-User-Id"
	HeaderUserRole = "X-User-Role"
)

func RequireUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := strings.TrimSpace(c.GetHeader(HeaderUserID))
		if uid == "" {
			// Allow query param fallback (useful for EventSource which can't set headers).
			uid = strings.TrimSpace(c.Query("user_id"))
		}
		if uid == "" {
			// Browser EventSource cannot set custom headers; allow cookie fallback.
			if v, err := c.Cookie("user_id"); err == nil {
				uid = strings.TrimSpace(v)
			}
		}
		if uid == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing X-User-Id"})
			return
		}
		c.Set("user_id", uid)
		c.Next()
	}
}

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := strings.TrimSpace(c.GetHeader(HeaderUserRole))
		if role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin role required"})
			return
		}
		c.Next()
	}
}

func UserID(c *gin.Context) string {
	v, _ := c.Get("user_id")
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

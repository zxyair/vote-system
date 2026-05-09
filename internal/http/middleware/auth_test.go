package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func runAuthRequest(req *http.Request) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequireUser())
	r.GET("/protected", func(c *gin.Context) {
		c.String(http.StatusOK, UserID(c))
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestRequireUserRejectsMissingUser(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := runAuthRequest(req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestRequireUserAcceptsHeaderQueryAndCookie(t *testing.T) {
	tests := []struct {
		name string
		req  *http.Request
		want string
	}{
		{
			name: "header",
			req: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/protected", nil)
				req.Header.Set(HeaderUserID, "user_header")
				return req
			}(),
			want: "user_header",
		},
		{
			name: "query",
			req:  httptest.NewRequest(http.MethodGet, "/protected?user_id=user_query", nil),
			want: "user_query",
		},
		{
			name: "cookie",
			req: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/protected", nil)
				req.AddCookie(&http.Cookie{Name: "user_id", Value: "user_cookie"})
				return req
			}(),
			want: "user_cookie",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := runAuthRequest(tt.req)
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d body=%q", w.Code, http.StatusOK, w.Body.String())
			}
			if got := w.Body.String(); got != tt.want {
				t.Fatalf("body = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRequireAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequireAdmin())
	r.GET("/admin", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin", nil))
	if w.Code != http.StatusForbidden {
		t.Fatalf("missing role status = %d, want %d", w.Code, http.StatusForbidden)
	}

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set(HeaderUserRole, "admin")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("admin role status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

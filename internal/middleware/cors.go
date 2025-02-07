package middleware

import (
	"net/http"
)

type CORSMiddleware struct {
	allowedOrigins []string
}

func NewCORSMiddleware(origins []string) *CORSMiddleware {
	// 如果没有指定源，默认允许所有源
	if len(origins) == 0 {
		origins = []string{"*"}
	}
	return &CORSMiddleware{
		allowedOrigins: origins,
	}
}

func (m *CORSMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// 检查是否允许该源
		allowOrigin := "*"
		if origin != "" {
			allowed := false
			for _, allowedOrigin := range m.allowedOrigins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					allowed = true
					allowOrigin = origin
					break
				}
			}
			if !allowed {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
		}

		// 设置 CORS 头
		w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "3600")

		// 处理预检请求
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

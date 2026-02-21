package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
)

var (
	sessions = make(map[string]int)
	mu       sync.RWMutex
)

func SetSession(token string, userID int) {
	mu.Lock()
	defer mu.Unlock()
	sessions[token] = userID
}

func DeleteSession(token string) {
	mu.Lock()
	defer mu.Unlock()
	delete(sessions, token)
}

func Auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		token = strings.TrimPrefix(token, "Bearer ")

		mu.RLock()
		userID, exists := sessions[token]
		mu.RUnlock()

		if !exists {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		r.Header.Set("X-User-ID", strconv.Itoa(userID))
		next(w, r)
	}
}
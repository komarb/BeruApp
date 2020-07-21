package server

import (
	"beruAPI/logging"
	"net/http"
)

// authMiddleware - промежуточная функция, проверяющая все запросы к приложению на наличие
// авторизационного токена
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := logging.NewStatusWriter(w)
		sw.Header().Set("Content-Type", "application/json")
		if r.Header.Get("Authorization") != cfg.Beru.ApiToken {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		next.ServeHTTP(sw, r)
		logging.LogHandler(sw, r)
	})
}
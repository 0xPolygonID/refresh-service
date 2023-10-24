package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/0xPolygonID/refresh-service/logger"
	"github.com/go-chi/chi/v5/middleware"
)

func zapContextLogger(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		t1 := time.Now()
		defer func() {
			logger.DefaultLogger.Infow("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"remoteAddr", r.RemoteAddr,
				"responseTime", fmt.Sprintf("%d ms", time.Since(t1).Milliseconds()),
				"status", ww.Status())
			logger.DefaultLogger.Sync()
		}()

		next.ServeHTTP(ww, r)
	}
	return http.HandlerFunc(fn)
}

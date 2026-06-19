// Package server wires the gin HTTP router serving the API hooks and GUI.
package server

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/bonukr/phantom-http-server/internal/logbuffer"
	"github.com/bonukr/phantom-http-server/internal/settings"
)

// Server holds dependencies shared by the HTTP handlers.
type Server struct {
	cfg    *settings.Settings
	log    *slog.Logger
	buf    *logbuffer.Buffer
	stats  *Stats
	web    fs.FS
	engine *gin.Engine
}

// New builds a Server and configures all routes. webFS holds the GUI assets.
func New(cfg *settings.Settings, log *slog.Logger, webFS fs.FS) *Server {
	gin.SetMode(gin.ReleaseMode)
	s := &Server{
		cfg:   cfg,
		log:   log,
		buf:   logbuffer.New(500),
		stats: NewStats(),
		web:   webFS,
	}

	r := gin.New()
	r.Use(gin.Recovery(), s.requestLogger())
	s.routes(r)
	s.engine = r
	return s
}

func (s *Server) routes(r *gin.Engine) {
	r.GET("/", s.serveIndex)
	r.StaticFS("/static", http.FS(s.web))

	api := r.Group("/api")
	{
		api.GET("/status", s.handleStatus)
		api.GET("/stats", s.handleStats)
		api.GET("/apis", s.handleAPIs)
		api.GET("/logs", s.handleLogs)
		api.DELETE("/logs", s.handleClearLogs)
		api.GET("/logs/stream", s.handleLogStream)
	}

	for _, ep := range s.cfg.APIs {
		s.registerAPI(r, ep)
	}
}

// requestLogger is a gin middleware that logs each request via slog.
func (s *Server) requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		// Skip noisy internal polling except errors.
		if c.Request.URL.Path == "/api/logs" || c.Request.URL.Path == "/api/logs/stream" {
			return
		}
		s.log.Info("request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(start).Milliseconds(),
			"client", c.ClientIP(),
		)
	}
}

// Run starts the HTTP/HTTPS server and blocks until ctx is cancelled.
func (s *Server) Run(ctx context.Context) error {
	addr := formatAddr(s.cfg.Server.Port)
	srv := &http.Server{Addr: addr, Handler: s.engine}

	errCh := make(chan error, 1)
	go func() {
		scheme := "http"
		if s.cfg.Server.TLS.Enabled {
			scheme = "https"
		}
		s.log.Info("http server listening",
			"addr", addr,
			"scheme", scheme,
			"tls", s.cfg.Server.TLS.Enabled,
		)
		var err error
		if s.cfg.Server.TLS.Enabled {
			err = srv.ListenAndServeTLS(s.cfg.Server.TLS.CertFile, s.cfg.Server.TLS.KeyFile)
		} else {
			err = srv.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		s.log.Info("shutting down http server")
		return srv.Shutdown(shutdownCtx)
	}
}

func formatAddr(port int) string {
	return ":" + strconv.Itoa(port)
}

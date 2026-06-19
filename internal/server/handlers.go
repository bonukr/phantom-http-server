package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/bonukr/phantom-http-server/internal/logbuffer"
	"github.com/bonukr/phantom-http-server/internal/settings"
)

// ---- GUI ----

func (s *Server) serveIndex(c *gin.Context) {
	data, err := s.readWeb("index.html")
	if err != nil {
		c.String(http.StatusInternalServerError, "index not found")
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", data)
}

// ---- Monitoring ----

func (s *Server) handleStatus(c *gin.Context) {
	scheme := "http"
	if s.cfg.Server.TLS.Enabled {
		scheme = "https"
	}
	c.JSON(http.StatusOK, gin.H{
		"configOk":      true,
		"configSource":  "setting.yml",
		"scheme":        scheme,
		"port":          s.cfg.Server.Port,
		"tlsEnabled":    s.cfg.Server.TLS.Enabled,
		"apiCount":      len(s.cfg.APIs),
		"uptimeSeconds": s.stats.Snapshot(nil).UptimeSeconds,
	})
}

func (s *Server) handleStats(c *gin.Context) {
	perPath := make(map[string]int64)
	for _, e := range s.buf.List("", "", "", 10000) {
		perPath[e.Path]++
	}
	c.JSON(http.StatusOK, s.stats.Snapshot(perPath))
}

func (s *Server) handleAPIs(c *gin.Context) {
	out := make([]gin.H, 0, len(s.cfg.APIs))
	for _, api := range s.cfg.APIs {
		out = append(out, gin.H{
			"path":        api.Path,
			"methods":     api.Methods,
			"description": api.Description,
		})
	}
	c.JSON(http.StatusOK, out)
}

func (s *Server) handleLogs(c *gin.Context) {
	path := c.Query("path")
	method := c.Query("method")
	text := c.Query("q")
	limit := parseLimit(c.Query("limit"), 100)

	entries := s.buf.List(path, method, text, limit)
	if entries == nil {
		entries = []logbuffer.Entry{}
	}
	c.JSON(http.StatusOK, entries)
}

func (s *Server) handleClearLogs(c *gin.Context) {
	s.buf.Clear()
	s.stats.Reset()
	s.log.Info("request logs cleared", "client", c.ClientIP())
	c.Status(http.StatusNoContent)
}

func (s *Server) handleLogStream(c *gin.Context) {
	path := c.Query("path")
	method := c.Query("method")
	text := c.Query("q")

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	ch := s.buf.Subscribe()
	defer s.buf.Unsubscribe(ch)

	c.Stream(func(w io.Writer) bool {
		select {
		case entry, ok := <-ch:
			if !ok {
				return false
			}
			if path != "" && entry.Path != path {
				return true
			}
			if method != "" && entry.Method != method {
				return true
			}
			if text != "" && !matchesTextFilter(entry, text) {
				return true
			}
			data, err := json.Marshal(entry)
			if err != nil {
				return false
			}
			fmt.Fprintf(w, "event: log\ndata: %s\n\n", data)
			return true
		case <-time.After(25 * time.Second):
			fmt.Fprintf(w, ": keepalive\n\n")
			return true
		case <-c.Request.Context().Done():
			return false
		}
	})
}

// ---- Virtual API endpoints ----

func (s *Server) registerAPI(r *gin.Engine, ep settings.APIConfig) {
	methodSet := make(map[string]struct{}, len(ep.Methods))
	for _, m := range ep.Methods {
		methodSet[strings.ToUpper(m)] = struct{}{}
	}

	handler := func(c *gin.Context) {
		if _, ok := methodSet[c.Request.Method]; !ok {
			c.JSON(http.StatusMethodNotAllowed, gin.H{
				"error":   "method not allowed",
				"allowed": ep.Methods,
			})
			return
		}
		s.handleVirtualAPI(c, ep.Path)
	}

	r.Any(ep.Path, handler)
	r.Any(ep.Path+"/", handler)
}

func (s *Server) handleVirtualAPI(c *gin.Context, apiPath string) {
	bodyBytes, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
	if err != nil {
		s.log.Error("read request body failed", "path", apiPath, "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}

	headers := flattenHeaders(c.Request.Header)
	bodyStr := string(bodyBytes)

	entry := logbuffer.Entry{
		Time:     time.Now().UTC(),
		Path:     apiPath,
		Method:   c.Request.Method,
		ClientIP: c.ClientIP(),
		Query:    c.Request.URL.RawQuery,
		Headers:  headers,
		Body:     bodyStr,
		BodySize: len(bodyBytes),
		Status:   http.StatusOK,
	}

	entry = s.buf.Add(entry)
	s.stats.RecordRequest(apiPath)

	s.log.Info("api request",
		"api", apiPath,
		"method", entry.Method,
		"client", entry.ClientIP,
		"query", entry.Query,
		"body_size", entry.BodySize,
		"headers", headers,
		"body", truncateForLog(bodyStr, 4096),
	)

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "request received",
		"id":      entry.ID,
	})
}

func flattenHeaders(h http.Header) map[string]string {
	out := make(map[string]string, len(h))
	for k, vals := range h {
		out[k] = strings.Join(vals, ", ")
	}
	return out
}

func truncateForLog(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "...(truncated)"
}

func parseLimit(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	var n int
	if _, err := fmt.Sscan(raw, &n); err != nil || n < 1 {
		return fallback
	}
	if n > 1000 {
		return 1000
	}
	return n
}

func matchesTextFilter(e logbuffer.Entry, text string) bool {
	text = strings.ToLower(strings.TrimSpace(text))
	if text == "" {
		return true
	}
	blob := strings.ToLower(e.Body + " " + e.Query + " " + e.ClientIP)
	for k, v := range e.Headers {
		blob += " " + strings.ToLower(k) + " " + strings.ToLower(v)
	}
	return strings.Contains(blob, text)
}

func (s *Server) readWeb(name string) ([]byte, error) {
	return fs.ReadFile(s.web, name)
}

// PrettyBody returns indented JSON when possible.
func PrettyBody(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return body
	}
	if !bytes.HasPrefix([]byte(body), []byte("{")) && !bytes.HasPrefix([]byte(body), []byte("[")) {
		return body
	}
	var v any
	if err := json.Unmarshal([]byte(body), &v); err != nil {
		return body
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return body
	}
	return string(out)
}

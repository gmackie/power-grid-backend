package otel

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

var (
	endpoint    = envOr("OTEL_ENDPOINT", "https://otlp.forgegraf.com")
	serviceName = envOr("FG_APP", "power-grid")
	stage       = envOr("FG_STAGE", "production")
	disabled    = os.Getenv("OTEL_DISABLED") == "true"
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func Middleware(next http.Handler) http.Handler {
	if disabled {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(rw, r)
		go pushSpan(r.Method, r.URL.Path, rw.status, time.Since(start))
	})
}

func MiddlewareFunc(next http.HandlerFunc) http.HandlerFunc {
	if disabled {
		return next
	}
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: 200}
		next(rw, r)
		go pushSpan(r.Method, r.URL.Path, rw.status, time.Since(start))
	}
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func pushSpan(method, path string, status int, latency time.Duration) {
	traceID := randomHex(16)
	spanID := randomHex(8)
	now := time.Now().UnixNano()
	startNano := now - latency.Nanoseconds()
	latencyMs := latency.Milliseconds()

	statusCode := 1
	if status >= 500 {
		statusCode = 2
	}

	payload := map[string]interface{}{
		"resourceSpans": []map[string]interface{}{{
			"resource": map[string]interface{}{
				"attributes": []map[string]interface{}{
					{"key": "service.name", "value": map[string]string{"stringValue": serviceName}},
					{"key": "deployment.environment", "value": map[string]string{"stringValue": stage}},
				},
			},
			"scopeSpans": []map[string]interface{}{{
				"scope": map[string]string{"name": "@forgegraph/otel"},
				"spans": []map[string]interface{}{{
					"traceId":            traceID,
					"spanId":             spanID,
					"name":               fmt.Sprintf("%s %s", method, path),
					"kind":               2,
					"startTimeUnixNano":  fmt.Sprintf("%d", startNano),
					"endTimeUnixNano":    fmt.Sprintf("%d", now),
					"attributes": []map[string]interface{}{
						{"key": "http.method", "value": map[string]string{"stringValue": method}},
						{"key": "http.target", "value": map[string]string{"stringValue": path}},
						{"key": "http.status_code", "value": map[string]string{"intValue": fmt.Sprintf("%d", status)}},
						{"key": "http.latency_ms", "value": map[string]string{"intValue": fmt.Sprintf("%d", latencyMs)}},
					},
					"status": map[string]interface{}{"code": statusCode},
				}},
			}},
		}},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return
	}

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("POST", endpoint+"/v1/traces", bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err == nil {
		resp.Body.Close()
	}
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

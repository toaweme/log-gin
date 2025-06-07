package log

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	stdHttp "net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/toaweme/http"
	"github.com/toaweme/log"
)

// ResponseRecorder is used to capture the response body and status
type ResponseRecorder struct {
	gin.ResponseWriter
	body   *bytes.Buffer
	status int
}

// Write records the response body and writes to the underlying writer
func (r *ResponseRecorder) Write(data []byte) (int, error) {
	r.body.Write(data)
	return r.ResponseWriter.Write(data)
}

// WriteHeader captures the response status code
func (r *ResponseRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

// SlogMiddleware creates a Gin middleware for logging using slog
func SlogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// capture request details
		method := c.Request.Method
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		clientIP := c.ClientIP()

		// clone and read the request body (if needed for logging)
		var requestBody string
		if c.Request.Body != nil {
			bodyBytes, _ := io.ReadAll(c.Request.Body)
			requestBody = string(bodyBytes)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)) // Restore the body
		}

		// set up response recording
		recorder := &ResponseRecorder{
			ResponseWriter: c.Writer,
			body:           bytes.NewBuffer([]byte{}),
			status:         stdHttp.StatusOK,
		}
		c.Writer = recorder

		// process the request
		c.Next()

		// calculate response time
		latency := time.Since(start)

		// log request and response details
		Ctx(c).Info("api",
			"method", method,
			"path", path,
			"query", query,
			"client_ip", clientIP,
			"request_body", requestBody,
			"response_body", recorder.body.String(),
			"status", recorder.status,
			"latency", latency,
			"error", c.Errors.ByType(gin.ErrorTypePrivate).String(),
		)
	}
}

const (
	KeyLoggerRequestID = "id"
	KeyGinRequestID    = "id"
	KeyLoggerUserAgent = "user-agent"
	KeyGinUserAgent    = "user-agent"
)

func Ctx(reqCtx context.Context) *slog.Logger {
	return log.Logger.With(KeyLoggerRequestID, reqCtx.Value(KeyGinRequestID), KeyLoggerUserAgent, reqCtx.Value(KeyGinUserAgent))
}

func GinLogTraceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Request.Header.Get(http.ClientIDHeaderName)
		if id == "" {
			id = ID()
		}
		c.Set(KeyGinRequestID, id)

		client := c.Request.Header.Get(http.ClientUserAgentHeaderName)
		if client == "" {
			client = "unknown"
		}
		c.Set(KeyGinUserAgent, client)
	}
}

func ID() string {
	return uuid.New().String()
}

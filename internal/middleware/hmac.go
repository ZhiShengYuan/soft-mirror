package middleware

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// HMACMiddleware authenticates requests using HMAC-SHA256.
// Signing string: METHOD\nPATH\nTIMESTAMP\nNONCE\nSHA256(body)
type HMACMiddleware struct {
	secret   []byte
	maxDrift time.Duration
	nonces   sync.Map // nonce string -> expiry time.Time
}

// NewHMACMiddleware creates a new HMAC middleware with the given secret and drift window.
func NewHMACMiddleware(secret string, maxDrift time.Duration) *HMACMiddleware {
	return &HMACMiddleware{
		secret:   []byte(secret),
		maxDrift: maxDrift,
	}
}

// Middleware returns the Gin handler function.
func (h *HMACMiddleware) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := h.verify(c); err != nil {
			slog.Warn("HMAC auth failed", "error", err, "path", c.Request.URL.Path, "ip", c.ClientIP())
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.Next()
	}
}

func (h *HMACMiddleware) verify(c *gin.Context) error {
	// 1. Parse and validate timestamp
	tsStr := c.GetHeader("X-Timestamp")
	if tsStr == "" {
		return fmt.Errorf("missing X-Timestamp header")
	}
	tsUnix, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid X-Timestamp: %w", err)
	}
	ts := time.Unix(tsUnix, 0)
	drift := time.Since(ts)
	if drift < 0 {
		drift = -drift
	}
	if drift > h.maxDrift {
		return fmt.Errorf("timestamp drift too large: %v", drift)
	}

	// 2. Validate nonce
	nonce := c.GetHeader("X-Nonce")
	if nonce == "" {
		return fmt.Errorf("missing X-Nonce header")
	}
	now := time.Now()
	if expiry, loaded := h.nonces.Load(nonce); loaded {
		if now.Before(expiry.(time.Time)) {
			return fmt.Errorf("nonce already used (replay attack)")
		}
	}

	// 3. Read body (bounded), restore for downstream handlers
	bodyBytes, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, 512<<20))
	if err != nil {
		return fmt.Errorf("reading request body: %w", err)
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	// 4. Compute body hash
	bodyHash := sha256.Sum256(bodyBytes)
	bodyHashHex := hex.EncodeToString(bodyHash[:])

	// 5. Build string to sign
	stringToSign := fmt.Sprintf("%s\n%s\n%s\n%s\n%s",
		c.Request.Method,
		c.Request.URL.Path,
		tsStr,
		nonce,
		bodyHashHex,
	)

	// 6. Compute expected HMAC
	mac := hmac.New(sha256.New, h.secret)
	mac.Write([]byte(stringToSign))
	expected := mac.Sum(nil)

	// 7. Parse Authorization header
	authHeader := c.GetHeader("Authorization")
	if len(authHeader) < 6 || authHeader[:5] != "HMAC " {
		return fmt.Errorf("invalid Authorization header format")
	}
	sigHex := authHeader[5:]
	sig, err := hex.DecodeString(sigHex)
	if err != nil {
		return fmt.Errorf("invalid signature encoding: %w", err)
	}

	// 8. Constant-time comparison
	if !hmac.Equal(sig, expected) {
		return fmt.Errorf("signature mismatch")
	}

	// 9. Store nonce
	h.nonces.Store(nonce, now.Add(h.maxDrift))

	return nil
}

// StartCleanup starts a background goroutine that removes expired nonces every drift/2.
func (h *HMACMiddleware) StartCleanup(ctx context.Context) {
	interval := h.maxDrift / 2
	if interval < time.Second {
		interval = time.Second
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				now := time.Now()
				h.nonces.Range(func(key, value any) bool {
					if now.After(value.(time.Time)) {
						h.nonces.Delete(key)
					}
					return true
				})
			}
		}
	}()
}

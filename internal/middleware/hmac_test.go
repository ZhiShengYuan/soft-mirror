package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func signRequest(t *testing.T, secret, method, path, body string) (sig, ts, nonce string) {
	t.Helper()
	ts = strconv.FormatInt(time.Now().Unix(), 10)
	nonce = fmt.Sprintf("nonce-%d", time.Now().UnixNano())

	bodyHash := sha256.Sum256([]byte(body))
	bodyHashHex := hex.EncodeToString(bodyHash[:])
	stringToSign := fmt.Sprintf("%s\n%s\n%s\n%s\n%s", method, path, ts, nonce, bodyHashHex)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(stringToSign))
	sig = hex.EncodeToString(mac.Sum(nil))
	return
}

func TestHMACValid(t *testing.T) {
	secret := "test-secret"
	m := NewHMACMiddleware(secret, 5*time.Minute)

	router := gin.New()
	router.PUT("/test", m.Middleware(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	body := "hello"
	sig, ts, nonce := signRequest(t, secret, "PUT", "/test", body)

	req := httptest.NewRequest("PUT", "/test", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "HMAC "+sig)
	req.Header.Set("X-Timestamp", ts)
	req.Header.Set("X-Nonce", nonce)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHMACInvalidSignature(t *testing.T) {
	secret := "test-secret"
	m := NewHMACMiddleware(secret, 5*time.Minute)

	router := gin.New()
	router.PUT("/test", m.Middleware(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	body := "hello"
	_, ts, nonce := signRequest(t, secret, "PUT", "/test", body)

	req := httptest.NewRequest("PUT", "/test", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "HMAC "+hex.EncodeToString(make([]byte, 32))) // wrong sig
	req.Header.Set("X-Timestamp", ts)
	req.Header.Set("X-Nonce", nonce)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestHMACExpiredTimestamp(t *testing.T) {
	secret := "test-secret"
	m := NewHMACMiddleware(secret, 5*time.Minute)

	router := gin.New()
	router.PUT("/test", m.Middleware(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	body := "hello"
	oldTS := strconv.FormatInt(time.Now().Add(-10*time.Minute).Unix(), 10)
	nonce := "unique-nonce-1"
	bodyHash := sha256.Sum256([]byte(body))
	bodyHashHex := hex.EncodeToString(bodyHash[:])
	stringToSign := fmt.Sprintf("PUT\n/test\n%s\n%s\n%s", oldTS, nonce, bodyHashHex)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(stringToSign))
	sig := hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest("PUT", "/test", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "HMAC "+sig)
	req.Header.Set("X-Timestamp", oldTS)
	req.Header.Set("X-Nonce", nonce)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for expired timestamp, got %d", w.Code)
	}
}

func TestHMACReplay(t *testing.T) {
	secret := "test-secret"
	m := NewHMACMiddleware(secret, 5*time.Minute)

	router := gin.New()
	router.PUT("/test", m.Middleware(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	body := "hello"
	sig, ts, nonce := signRequest(t, secret, "PUT", "/test", body)

	// First request — should succeed
	req1 := httptest.NewRequest("PUT", "/test", bytes.NewBufferString(body))
	req1.Header.Set("Authorization", "HMAC "+sig)
	req1.Header.Set("X-Timestamp", ts)
	req1.Header.Set("X-Nonce", nonce)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("first request failed: %d %s", w1.Code, w1.Body.String())
	}

	// Second request with same nonce — should be rejected as replay
	req2 := httptest.NewRequest("PUT", "/test", bytes.NewBufferString(body))
	req2.Header.Set("Authorization", "HMAC "+sig)
	req2.Header.Set("X-Timestamp", ts)
	req2.Header.Set("X-Nonce", nonce)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for replay, got %d", w2.Code)
	}
}

func TestHMACMissingHeaders(t *testing.T) {
	secret := "test-secret"
	m := NewHMACMiddleware(secret, 5*time.Minute)

	router := gin.New()
	router.PUT("/test", m.Middleware(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("PUT", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for missing headers, got %d", w.Code)
	}
}

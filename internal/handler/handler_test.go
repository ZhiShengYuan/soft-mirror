package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"file-host/internal/handler"
	"file-host/internal/storage"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// newRouter builds a minimal test router wired to the given store.
func newRouter(store *storage.Store) *gin.Engine {
	r := gin.New()
	r.GET("/healthz", handler.Health)
	api := r.Group("/api/v1")
	api.GET("/programs/:name/versions", handler.Versions(store))
	api.GET("/programs/:name/download", handler.AutoDownload(store))
	api.GET("/programs/:name/:version/:os/:arch", handler.DirectDownload(store))
	api.PUT("/programs/:name/:version/:os/:arch", handler.Upload(store, 1<<20))
	api.DELETE("/programs/:name/:version/:os/:arch", handler.DeleteBinary(store))
	api.DELETE("/programs/:name/:version", handler.DeleteVersion(store))
	return r
}

// newStore creates an isolated store in a temp dir.
func newStore(t *testing.T) *storage.Store {
	t.Helper()
	store, err := storage.New(t.TempDir())
	if err != nil {
		t.Fatalf("storage.New: %v", err)
	}
	return store
}

// seedBinary uploads a tiny binary into the store.
func seedBinary(t *testing.T, store *storage.Store, program, version, osName, arch string) {
	t.Helper()
	if err := store.PutBinary(program, version, osName, arch, "", bytes.NewReader([]byte("binary")), 1<<20); err != nil {
		t.Fatalf("seed PutBinary: %v", err)
	}
}

func TestHealth(t *testing.T) {
	r := newRouter(newStore(t))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("parse body: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("expected status=ok, got %q", body["status"])
	}
}

func TestVersions_Empty(t *testing.T) {
	r := newRouter(newStore(t))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/programs/myapp/versions", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("parse body: %v", err)
	}
	versions := body["versions"].([]interface{})
	if len(versions) != 0 {
		t.Errorf("expected empty versions, got %d", len(versions))
	}
}

func TestVersions_WithData(t *testing.T) {
	store := newStore(t)
	seedBinary(t, store, "myapp", "v1.0.0", "linux", "amd64")
	seedBinary(t, store, "myapp", "v1.1.0", "linux", "amd64")

	r := newRouter(store)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/programs/myapp/versions", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("parse body: %v", err)
	}
	versions := body["versions"].([]interface{})
	if len(versions) != 2 {
		t.Errorf("expected 2 versions, got %d", len(versions))
	}
}

func TestVersions_InvalidName(t *testing.T) {
	r := newRouter(newStore(t))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/programs/../etc/versions", nil))

	// Gin normalises path — result must not be 200 with injected data
	if w.Code == http.StatusOK {
		t.Errorf("expected non-200 for path traversal attempt, got 200")
	}
}

func TestDirectDownload_NotFound(t *testing.T) {
	r := newRouter(newStore(t))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/programs/myapp/v1.0.0/linux/amd64", nil))

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestDirectDownload_Found(t *testing.T) {
	store := newStore(t)
	seedBinary(t, store, "myapp", "v1.0.0", "linux", "amd64")

	r := newRouter(store)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/programs/myapp/v1.0.0/linux/amd64", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "binary" {
		t.Errorf("body mismatch: %q", w.Body.String())
	}
	if w.Header().Get("ETag") == "" {
		t.Error("expected ETag header")
	}
	if w.Header().Get("Cache-Control") == "" {
		t.Error("expected Cache-Control header")
	}
	if w.Header().Get("Content-Disposition") == "" {
		t.Error("expected Content-Disposition header")
	}
}

func TestDirectDownload_WindowsExeFilename(t *testing.T) {
	store := newStore(t)
	seedBinary(t, store, "myapp", "v1.0.0", "windows", "amd64")

	r := newRouter(store)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/programs/myapp/v1.0.0/windows/amd64", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	cd := w.Header().Get("Content-Disposition")
	if cd == "" {
		t.Fatal("missing Content-Disposition")
	}
	// Should reference human-readable filename with .exe extension
	if cd != `attachment; filename="myapp-windows-amd64-v1.0.0.exe"` {
		t.Errorf("unexpected Content-Disposition: %q", cd)
	}
}

func TestDirectDownload_InvalidOS(t *testing.T) {
	r := newRouter(newStore(t))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/programs/myapp/v1.0.0/plan9/amd64", nil))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid OS, got %d", w.Code)
	}
}

func TestDirectDownload_InvalidArch(t *testing.T) {
	r := newRouter(newStore(t))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/programs/myapp/v1.0.0/linux/i386", nil))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid arch, got %d", w.Code)
	}
}

func TestDirectDownload_ConditionalGet_NotModified(t *testing.T) {
	store := newStore(t)
	seedBinary(t, store, "myapp", "v1.0.0", "linux", "amd64")

	r := newRouter(store)

	// First request to get the ETag
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, httptest.NewRequest(http.MethodGet, "/api/v1/programs/myapp/v1.0.0/linux/amd64", nil))
	if w1.Code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", w1.Code)
	}
	etag := w1.Header().Get("ETag")
	if etag == "" {
		t.Fatal("no ETag on first response")
	}

	// Second request with If-None-Match should yield 304
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/programs/myapp/v1.0.0/linux/amd64", nil)
	req2.Header.Set("If-None-Match", etag)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusNotModified {
		t.Errorf("conditional GET: expected 304, got %d", w2.Code)
	}
}

func TestAutoDownload_NotFound(t *testing.T) {
	r := newRouter(newStore(t))
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/programs/missing/download", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64)")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestAutoDownload_Redirect(t *testing.T) {
	store := newStore(t)
	seedBinary(t, store, "myapp", "v1.0.0", "linux", "amd64")

	r := newRouter(store)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/programs/myapp/download", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64)")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", w.Code)
	}
	loc := w.Header().Get("Location")
	if loc == "" {
		t.Error("expected Location header on redirect")
	}
}

func TestAutoDownload_QueryOverride(t *testing.T) {
	store := newStore(t)
	seedBinary(t, store, "myapp", "v1.0.0", "darwin", "arm64")

	r := newRouter(store)
	w := httptest.NewRecorder()
	// Send linux UA but override to darwin/arm64 via query params
	req := httptest.NewRequest(http.MethodGet, "/api/v1/programs/myapp/download?os=darwin&arch=arm64", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64)")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", w.Code)
	}
	loc := w.Header().Get("Location")
	if loc != "/api/v1/programs/myapp/v1.0.0/darwin/arm64" {
		t.Errorf("unexpected redirect location: %q", loc)
	}
}

func TestAutoDownload_InvalidOSQuery(t *testing.T) {
	r := newRouter(newStore(t))
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/programs/myapp/download?os=plan9", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid OS query, got %d", w.Code)
	}
}

func TestUpload_Success(t *testing.T) {
	r := newRouter(newStore(t))
	w := httptest.NewRecorder()
	body := bytes.NewReader([]byte("exe content"))
	req := httptest.NewRequest(http.MethodPut, "/api/v1/programs/myapp/v1.0.0/linux/amd64", body)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if resp["program"] != "myapp" {
		t.Errorf("program mismatch: %q", resp["program"])
	}
}

func TestUpload_InvalidVersion(t *testing.T) {
	r := newRouter(newStore(t))
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/programs/myapp/not-a-version/linux/amd64", bytes.NewReader([]byte("x")))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid version, got %d", w.Code)
	}
}

func TestDeleteBinary_Success(t *testing.T) {
	store := newStore(t)
	seedBinary(t, store, "myapp", "v1.0.0", "linux", "amd64")

	r := newRouter(store)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/api/v1/programs/myapp/v1.0.0/linux/amd64", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify it's gone
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/api/v1/programs/myapp/v1.0.0/linux/amd64", nil))
	if w2.Code != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", w2.Code)
	}
}

func TestDeleteVersion_Success(t *testing.T) {
	store := newStore(t)
	seedBinary(t, store, "myapp", "v1.0.0", "linux", "amd64")
	seedBinary(t, store, "myapp", "v1.0.0", "darwin", "arm64")

	r := newRouter(store)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/api/v1/programs/myapp/v1.0.0", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify versions is now empty
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/api/v1/programs/myapp/versions", nil))
	var body map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &body)
	versions := body["versions"].([]interface{})
	if len(versions) != 0 {
		t.Errorf("expected no versions after delete, got %d", len(versions))
	}
}

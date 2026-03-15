package handler

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"file-host/internal/detect"
	"file-host/internal/semver"
	"file-host/internal/storage"
	"file-host/internal/validate"
)

// AutoDownload handles GET /api/v1/programs/:name/download
// Detects OS/arch from User-Agent (overridable via query params), resolves version, and redirects.
func AutoDownload(store *storage.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		if err := validate.ValidateProgramName(name); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Detect OS/arch from User-Agent, allow query param overrides
		detectedOS, detectedArch := detect.Detect(c.Request.UserAgent())
		osName := c.DefaultQuery("os", detectedOS)
		arch := c.DefaultQuery("arch", detectedArch)
		versionQuery := c.DefaultQuery("version", "latest")

		// Validate OS/arch if explicitly provided
		if err := validate.ValidateOS(osName); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := validate.ValidateArch(arch); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// List available versions
		available, err := store.ListVersions(name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list versions"})
			return
		}
		if len(available) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "program not found"})
			return
		}

		// Resolve version
		resolved, err := semver.Resolve(versionQuery, available)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("version not found: %s", err.Error())})
			return
		}

		// Verify the binary exists for the requested platform
		if _, err := store.GetBinaryPath(name, resolved, osName, arch); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("binary not found for %s/%s", osName, arch)})
			return
		}

		// 302 redirect to direct download URL
		target := fmt.Sprintf("/api/v1/programs/%s/%s/%s/%s", name, resolved, osName, arch)
		c.Header("Cache-Control", "no-cache")
		c.Redirect(http.StatusFound, target)
	}
}

// DirectDownload handles GET /api/v1/programs/:name/:version/:os/:arch
// Serves the binary file with ETag, Range, and cache support.
func DirectDownload(store *storage.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		version := c.Param("version")
		osName := c.Param("os")
		arch := c.Param("arch")

		if err := validate.ValidateProgramName(name); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := validate.ValidateOS(osName); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := validate.ValidateArch(arch); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := validate.ValidateVersion(version); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Resolve version to the on-disk directory name (handles v-prefix mismatch)
		available, err := store.ListVersions(name)
		if err != nil || len(available) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "program not found"})
			return
		}
		resolved, err := semver.Resolve(version, available)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("version not found: %s", err.Error())})
			return
		}
		version = resolved

		info, err := store.BinaryInfo(name, version, osName, arch)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "binary not found"})
			return
		}

		path, err := store.GetBinaryPath(name, version, osName, arch)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "binary not found"})
			return
		}

		f, err := os.Open(path)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open file"})
			return
		}
		defer f.Close()

		// Set headers
		etag := fmt.Sprintf(`"%x-%x"`, info.ModTime.Unix(), info.Size)
		c.Header("ETag", etag)
		c.Header("Cache-Control", "public, max-age=86400")

		// Use the actual stored filename for the download
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, info.Filename))

		http.ServeContent(c.Writer, c.Request, info.Filename, info.ModTime, f)
	}
}

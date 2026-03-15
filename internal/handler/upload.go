package handler

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"

	"file-host/internal/storage"
	"file-host/internal/validate"
)

var extensionRegex = regexp.MustCompile(`^\.[a-zA-Z0-9]{1,16}$`)

// Upload handles PUT /api/v1/programs/:name/:version/:os/:arch
// Stores the raw request body as the binary.
func Upload(store *storage.Store, maxSize int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		version := c.Param("version")
		osName := c.Param("os")
		arch := c.Param("arch")

		if err := validate.ValidateProgramName(name); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := validate.ValidateVersion(version); err != nil {
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

		// Determine filename: respect X-Extension header if provided and valid.
		var fname string
		if ext := c.GetHeader("X-Extension"); ext != "" {
			if ext[0] != '.' {
				ext = "." + ext
			}
			if !extensionRegex.MatchString(ext) {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid X-Extension %q: must match %s", ext, extensionRegex)})
				return
			}
			fname = name + ext
		}

		if err := store.PutBinary(name, version, osName, arch, fname, c.Request.Body, maxSize); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "uploaded successfully",
			"program": name,
			"version": version,
			"os":      osName,
			"arch":    arch,
		})
	}
}

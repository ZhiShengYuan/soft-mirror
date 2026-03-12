package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"file-host/internal/storage"
	"file-host/internal/validate"
)

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

		if err := store.PutBinary(name, version, osName, arch, c.Request.Body, maxSize); err != nil {
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

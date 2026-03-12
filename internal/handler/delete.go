package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"file-host/internal/storage"
	"file-host/internal/validate"
)

// DeleteBinary handles DELETE /api/v1/programs/:name/:version/:os/:arch
func DeleteBinary(store *storage.Store) gin.HandlerFunc {
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

		if err := store.DeleteBinary(name, version, osName, arch); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "deleted"})
	}
}

// DeleteVersion handles DELETE /api/v1/programs/:name/:version
func DeleteVersion(store *storage.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		version := c.Param("version")

		if err := validate.ValidateProgramName(name); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := validate.ValidateVersion(version); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := store.DeleteVersion(name, version); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "version deleted"})
	}
}

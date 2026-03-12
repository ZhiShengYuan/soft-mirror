package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"file-host/internal/model"
	"file-host/internal/storage"
	"file-host/internal/validate"
)

type versionInfo struct {
	Version   string           `json:"version"`
	Platforms []model.Platform `json:"platforms"`
}

type versionsResponse struct {
	Program  string        `json:"program"`
	Versions []versionInfo `json:"versions"`
}

// Versions handles GET /api/v1/programs/:name/versions
func Versions(store *storage.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		if err := validate.ValidateProgramName(name); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		versions, err := store.ListVersions(name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list versions"})
			return
		}

		var versionInfos []versionInfo
		for _, v := range versions {
			platforms, err := store.ListPlatforms(name, v)
			if err != nil {
				continue
			}
			versionInfos = append(versionInfos, versionInfo{
				Version:   v,
				Platforms: platforms,
			})
		}

		if versionInfos == nil {
			versionInfos = []versionInfo{}
		}

		c.JSON(http.StatusOK, versionsResponse{
			Program:  name,
			Versions: versionInfos,
		})
	}
}

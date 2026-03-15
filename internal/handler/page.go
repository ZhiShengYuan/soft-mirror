package handler

import (
	"fmt"
	"html/template"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"

	semverlib "github.com/Masterminds/semver/v3"

	filesemver "file-host/internal/semver"
	"file-host/internal/storage"
	"file-host/internal/validate"
)

// PlatformData holds display info for a single platform.
type PlatformData struct {
	OS          string
	Arch        string
	SizeFormatted string
	DownloadURL string
}

// VersionData holds display info for a single version.
type VersionData struct {
	Version   string
	Platforms []PlatformData
}

// ProgramPageData is the template data for the program detail page.
type ProgramPageData struct {
	Program  string
	Versions []VersionData
}

// ProgramPage handles GET /programs/:name
func ProgramPage(store *storage.Store, tmpl *template.Template) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		if err := validate.ValidateProgramName(name); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		rawVersions, err := store.ListVersions(name)
		if err != nil || len(rawVersions) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "program not found"})
			return
		}

		// Sort versions newest first
		parsed := make([]*semverlib.Version, 0, len(rawVersions))
		verMap := map[string]string{} // canonical -> original
		for _, v := range rawVersions {
			sv, err := filesemver.CoerceVersion(v)
			if err == nil {
				parsed = append(parsed, sv)
				verMap[sv.Original()] = v
			}
		}
		sort.Slice(parsed, func(i, j int) bool {
			return parsed[i].GreaterThan(parsed[j])
		})

		var versions []VersionData
		for _, sv := range parsed {
			orig := verMap[sv.Original()]
			platforms, err := store.ListPlatforms(name, orig)
			if err != nil {
				continue
			}
			var platData []PlatformData
			for _, p := range platforms {
				info, err := store.BinaryInfo(name, orig, p.OS, p.Arch)
				sizeStr := "unknown"
				if err == nil {
					sizeStr = formatSize(info.Size)
				}
				platData = append(platData, PlatformData{
					OS:            p.OS,
					Arch:          p.Arch,
					SizeFormatted: sizeStr,
					DownloadURL:   fmt.Sprintf("/api/v1/programs/%s/%s/%s/%s", name, orig, p.OS, p.Arch),
				})
			}
			versions = append(versions, VersionData{
				Version:   orig,
				Platforms: platData,
			})
		}

		data := ProgramPageData{
			Program:  name,
			Versions: versions,
		}

		c.Header("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(c.Writer, "program.html", data); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "template error"})
		}
	}
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

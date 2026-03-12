package handler

import (
	"fmt"
	"html/template"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"

	semverlib "github.com/Masterminds/semver/v3"

	"file-host/internal/storage"
)

// ProgramSummary holds display info for a program on the index page.
type ProgramSummary struct {
	Name          string
	LatestVersion string
	VersionCount  int
	Platforms     []string // e.g. "linux/amd64"
}

// IndexData is the template data for the index page.
type IndexData struct {
	Programs []ProgramSummary
}

// Index handles GET /
func Index(store *storage.Store, tmpl *template.Template) gin.HandlerFunc {
	return func(c *gin.Context) {
		programs, err := store.ListPrograms()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list programs"})
			return
		}

		var summaries []ProgramSummary
		for _, prog := range programs {
			versions, err := store.ListVersions(prog)
			if err != nil || len(versions) == 0 {
				summaries = append(summaries, ProgramSummary{Name: prog})
				continue
			}

			// Find latest non-prerelease version
			var parsed []*semverlib.Version
			for _, v := range versions {
				if sv, err := semverlib.NewVersion(v); err == nil {
					parsed = append(parsed, sv)
				}
			}
			sort.Slice(parsed, func(i, j int) bool {
				return parsed[i].GreaterThan(parsed[j])
			})

			latest := ""
			for _, sv := range parsed {
				if sv.Prerelease() == "" {
					latest = sv.Original()
					break
				}
			}
			if latest == "" && len(parsed) > 0 {
				latest = parsed[0].Original()
			}

			// Collect unique platforms across all versions (from latest)
			platformSet := map[string]bool{}
			if latest != "" {
				platforms, _ := store.ListPlatforms(prog, latest)
				for _, p := range platforms {
					platformSet[fmt.Sprintf("%s/%s", p.OS, p.Arch)] = true
				}
			}
			var platformList []string
			for k := range platformSet {
				platformList = append(platformList, k)
			}
			sort.Strings(platformList)

			summaries = append(summaries, ProgramSummary{
				Name:          prog,
				LatestVersion: latest,
				VersionCount:  len(versions),
				Platforms:     platformList,
			})
		}

		if summaries == nil {
			summaries = []ProgramSummary{}
		}

		c.Header("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(c.Writer, "index.html", IndexData{Programs: summaries}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "template error"})
		}
	}
}

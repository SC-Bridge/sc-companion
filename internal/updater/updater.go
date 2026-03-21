package updater

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	repoOwner = "SC-Bridge"
	repoName  = "sc-companion"
	apiURL    = "https://api.github.com/repos/" + repoOwner + "/" + repoName + "/releases/latest"
)

// ReleaseInfo contains version info from GitHub Releases.
type ReleaseInfo struct {
	Version     string `json:"version"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	DownloadURL string `json:"downloadUrl"`
	PublishedAt string `json:"publishedAt"`
	HasUpdate   bool   `json:"hasUpdate"`
}

type ghRelease struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	HTMLURL     string    `json:"html_url"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// CheckForUpdate checks GitHub Releases for a newer version.
func CheckForUpdate(currentVersion string) (*ReleaseInfo, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "SCBridgeCompanion/"+currentVersion)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		// No releases yet
		return &ReleaseInfo{Version: currentVersion, HasUpdate: false}, nil
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("github api returned %d", resp.StatusCode)
	}

	var release ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")

	// Find the Windows zip asset
	var downloadURL string
	for _, asset := range release.Assets {
		if strings.Contains(asset.Name, "windows") && strings.HasSuffix(asset.Name, ".zip") {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}
	// Fall back to installer
	if downloadURL == "" {
		for _, asset := range release.Assets {
			if strings.HasSuffix(asset.Name, "-installer.exe") {
				downloadURL = asset.BrowserDownloadURL
				break
			}
		}
	}

	return &ReleaseInfo{
		Version:     latestVersion,
		Name:        release.Name,
		URL:         release.HTMLURL,
		DownloadURL: downloadURL,
		PublishedAt: release.PublishedAt.Format(time.RFC3339),
		HasUpdate:   compareVersions(latestVersion, currentVersion),
	}, nil
}

// compareVersions returns true if latest > current (simple semver comparison).
func compareVersions(latest, current string) bool {
	lParts := parseVersion(latest)
	cParts := parseVersion(current)

	for i := 0; i < 3; i++ {
		if lParts[i] > cParts[i] {
			return true
		}
		if lParts[i] < cParts[i] {
			return false
		}
	}
	return false
}

func parseVersion(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	var result [3]int
	for i, p := range parts {
		if i >= 3 {
			break
		}
		// Parse only digits, ignore pre-release suffixes
		n := 0
		for _, c := range p {
			if c >= '0' && c <= '9' {
				n = n*10 + int(c-'0')
			} else {
				break
			}
		}
		result[i] = n
	}
	return result
}

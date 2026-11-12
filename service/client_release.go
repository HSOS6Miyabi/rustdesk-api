package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	officialRustDeskReleasesURL = "https://api.github.com/repos/rustdesk/rustdesk/releases"
	maxClientInstallerSize int64 = 100 << 20
)

var officialClientVersionPattern = regexp.MustCompile(`^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,4}$`)

func validOfficialClientVersion(version string) bool {
	return len(version) <= 30 && officialClientVersionPattern.MatchString(version)
}

type officialRustDeskAsset struct {
	Name string `json:"name"`
	Size int64 `json:"size"`
	Digest string `json:"digest"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type officialRustDeskRelease struct {
	TagName string `json:"tag_name"`
	Draft bool `json:"draft"`
	Prerelease bool `json:"prerelease"`
	Assets []officialRustDeskAsset `json:"assets"`
}

type OfficialClientVersion struct {
	Version string `json:"version"`
	Platforms []string `json:"platforms"`
}

var officialClientPlatforms = []string{
	"windows-x64", "windows-arm64",
	"linux-x64", "linux-arm64",
	"macos-x64", "macos-arm64",
}

func officialClientAssetName(platform, version string) (string, error) {
	if !validOfficialClientVersion(version) {
		return "", errors.New("invalid official RustDesk version")
	}
	architecture, extension := "", ""
	switch platform {
	case "windows-x64":
		architecture, extension = "x86_64", ".exe"
	case "windows-arm64":
		architecture, extension = "aarch64", ".exe"
	case "linux-x64":
		architecture, extension = "x86_64", ".deb"
	case "linux-arm64":
		architecture, extension = "aarch64", ".deb"
	case "macos-x64":
		architecture, extension = "x86_64", ".dmg"
	case "macos-arm64":
		architecture, extension = "aarch64", ".dmg"
	default:
		return "", errors.New("unsupported client platform")
	}
	return "rustdesk-" + version + "-" + architecture + extension, nil
}

func officialSourceFromRelease(platform, version string, release officialRustDeskRelease) (clientBuildSource, error) {
	name, err := officialClientAssetName(platform, version)
	if err != nil {
		return clientBuildSource{}, err
	}
	if release.Draft || release.Prerelease || release.TagName != version {
		return clientBuildSource{}, errors.New("official RustDesk release is unavailable or not stable")
	}
	wantURL := "https://github.com/rustdesk/rustdesk/releases/download/" + url.PathEscape(version) + "/" + name
	for _, asset := range release.Assets {
		if asset.Name != name {
			continue
		}
		if asset.BrowserDownloadURL != wantURL ||
			asset.Size <= 0 || asset.Size > maxClientInstallerSize {
			return clientBuildSource{}, errors.New("official installer metadata is invalid")
		}
		digest := ""
		if asset.Digest != "" {
			if !strings.HasPrefix(asset.Digest, "sha256:") {
				return clientBuildSource{}, errors.New("official installer uses an unsupported digest")
			}
			digest = strings.TrimPrefix(asset.Digest, "sha256:")
			if len(digest) != 64 {
				return clientBuildSource{}, errors.New("official installer SHA-256 is invalid")
			}
			for _, c := range digest {
				if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
					return clientBuildSource{}, errors.New("official installer SHA-256 is invalid")
				}
			}
		}
		return clientBuildSource{URL: wantURL, SHA256: digest, Size: asset.Size}, nil
	}
	return clientBuildSource{}, fmt.Errorf("official RustDesk %s has no installer for %s", version, platform)
}

func officialMetadataClient() *http.Client {
	client := buildClientHTTP()
	client.Timeout = 8 * time.Second
	return client
}

func fetchOfficialReleaseJSON(ctx context.Context, client *http.Client, endpoint string, target interface{}) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "rustdesk-api-client-installer")
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("cannot retrieve official RustDesk releases: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("official RustDesk release lookup returned HTTP %d", response.StatusCode)
	}
	const limit = 5 << 20
	data, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return err
	}
	if len(data) > limit {
		return errors.New("official release metadata is too large")
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("cannot decode official RustDesk release information: %w", err)
	}
	return nil
}

func officialClientBuildSourceWithURL(ctx context.Context, client *http.Client, baseURL, platform, version string) (clientBuildSource, error) {
	if _, err := officialClientAssetName(platform, version); err != nil {
		return clientBuildSource{}, err
	}
	var release officialRustDeskRelease
	if err := fetchOfficialReleaseJSON(ctx, client, baseURL+"/tags/"+url.PathEscape(version), &release); err != nil {
		return clientBuildSource{}, err
	}
	return officialSourceFromRelease(platform, version, release)
}

func officialClientBuildSource(platform, version string) (clientBuildSource, error) {
	return officialClientBuildSourceWithURL(context.Background(), officialMetadataClient(),
		officialRustDeskReleasesURL, platform, version)
}

func versionsFromOfficialReleases(releases []officialRustDeskRelease) []OfficialClientVersion {
	versions := make([]OfficialClientVersion, 0, len(releases))
	seen := make(map[string]bool, len(releases))
	for _, release := range releases {
		version := release.TagName
		if !validOfficialClientVersion(version) || release.Draft || release.Prerelease || seen[version] {
			continue
		}
		seen[version] = true
		item := OfficialClientVersion{Version: version}
		for _, platform := range officialClientPlatforms {
			if _, err := officialSourceFromRelease(platform, version, release); err == nil {
				item.Platforms = append(item.Platforms, platform)
			}
		}
		if len(item.Platforms) != 0 {
			versions = append(versions, item)
		}
	}
	return versions
}

var clientVersionCache struct {
	sync.Mutex
	until time.Time
	items []OfficialClientVersion
}

func ListOfficialClientVersions(ctx context.Context) ([]OfficialClientVersion, error) {
	clientVersionCache.Lock()
	if time.Now().Before(clientVersionCache.until) {
		result := append([]OfficialClientVersion(nil), clientVersionCache.items...)
		clientVersionCache.Unlock()
		return result, nil
	}
	clientVersionCache.Unlock()
	var releases []officialRustDeskRelease
	if err := fetchOfficialReleaseJSON(ctx, officialMetadataClient(),
		officialRustDeskReleasesURL+"?per_page=30", &releases); err != nil {
		return nil, err
	}
	result := versionsFromOfficialReleases(releases)
	clientVersionCache.Lock()
	clientVersionCache.until = time.Now().Add(15 * time.Minute)
	clientVersionCache.items = result
	clientVersionCache.Unlock()
	return result, nil
}

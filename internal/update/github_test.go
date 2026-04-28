package update

import (
	"io"
	"net/http"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestFetchLatestRelease_OK(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			require.Equal(t, http.MethodGet, req.Method)
			require.Equal(t, "https://example.com/repos/api7/a7/releases/latest", req.URL.String())
			assert.Equal(t, "application/vnd.github.v3+json", req.Header.Get("Accept"))
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body: io.NopCloser(strings.NewReader(`{
					"tag_name": "v1.2.3",
					"name": "v1.2.3",
					"body": "release notes",
					"html_url": "https://github.com/api7/a7/releases/tag/v1.2.3",
					"assets": [{
						"name": "a7_1.2.3_linux_amd64.tar.gz",
						"browser_download_url": "https://example.com/a7.tar.gz",
						"size": 123,
						"content_type": "application/gzip"
					}]
				}`)),
			}, nil
		}),
	}

	release, err := fetchLatestRelease("https://example.com", client)
	require.NoError(t, err)
	assert.Equal(t, "v1.2.3", release.TagName)
	assert.Equal(t, "v1.2.3", release.Name)
	assert.Len(t, release.Assets, 1)
	assert.Equal(t, "a7_1.2.3_linux_amd64.tar.gz", release.Assets[0].Name)
}

func TestFetchLatestRelease_404ReturnsEmpty(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	release, err := fetchLatestRelease("https://example.com", client)
	require.NoError(t, err)
	assert.Equal(t, Release{}, release)
}

func TestFetchLatestRelease_BadStatus(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	_, err := fetchLatestRelease("https://example.com", client)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}

func TestFindAsset_MatchCurrentPlatform(t *testing.T) {
	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}
	current := "a7_1.0.0_" + runtime.GOOS + "_" + runtime.GOARCH + ext

	release := Release{
		TagName: "v1.0.0",
		Assets: []Asset{
			{Name: current},
			{Name: "a7_1.0.0_linux_amd64.tar.gz"},
			{Name: "a7_1.0.0_darwin_arm64.tar.gz"},
			{Name: "a7_1.0.0_windows_amd64.zip"},
		},
	}

	asset, err := FindAsset(release)
	require.NoError(t, err)
	assert.Equal(t, current, asset.Name)
}

func TestFindAsset_NoMatch(t *testing.T) {
	release := Release{
		TagName: "v1.0.0",
		Assets: []Asset{
			{Name: "a7_1.0.0_linux_386.tar.gz"},
		},
	}

	_, err := FindAsset(release)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no matching release asset")
}

package dispatch

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func newTestInstallModule() *Module {
	cfg := DefaultConfig()
	return &Module{
		logger: zap.NewNop(),
		cfg:    cfg,
	}
}

func TestHandleInstallScript(t *testing.T) {
	tests := []struct {
		name             string
		platform         string
		arch             string
		token            string
		wantStatus       int
		wantContains     []string
		wantNotContains  []string
		wantDisposition  string
	}{
		{
			name:     "Linux",
			platform: "linux",
			arch:     "amd64",
			token:    "test-enroll-token-123",
			wantStatus: http.StatusOK,
			wantContains: []string{
				"#!/usr/bin/env bash",
				"test-enroll-token-123",
				"9090",
				"systemctl",
				"systemd",
			},
			wantDisposition: ".sh",
		},
		{
			name:     "Windows",
			platform: "windows",
			arch:     "amd64",
			token:    "test-enroll-token-456",
			wantStatus: http.StatusOK,
			wantContains: []string{
				"#Requires -RunAsAdministrator",
				"$ErrorActionPreference",
				"test-enroll-token-456",
				"scout.exe",
				"New-Service",
			},
			wantDisposition: ".ps1",
		},
		{
			name:     "MacOS",
			platform: "darwin",
			arch:     "arm64",
			token:    "test-enroll-token-789",
			wantStatus: http.StatusOK,
			wantContains: []string{
				"#!/usr/bin/env bash",
				"test-enroll-token-789",
				"9090",
			},
			wantNotContains: []string{
				"systemctl",
				"systemd",
			},
			wantDisposition: ".sh",
		},
		{
			name:       "InvalidPlatform",
			platform:   "freebsd",
			arch:       "amd64",
			token:      "some-token",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "MissingToken",
			platform:   "linux",
			arch:       "amd64",
			token:      "",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "InvalidArch",
			platform:   "windows",
			arch:       "arm64",
			token:      "some-token",
			wantStatus: http.StatusBadRequest,
		},
	}

	m := newTestInstallModule()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := "/dispatch/install/" + tc.platform + "/" + tc.arch
			if tc.token != "" {
				path += "?token=" + tc.token
			}
			req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
			req.SetPathValue("platform", tc.platform)
			req.SetPathValue("arch", tc.arch)
			req.Host = "192.168.1.100:8080"

			rec := httptest.NewRecorder()
			m.handleInstallScript(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}

			if tc.wantStatus != http.StatusOK {
				return
			}

			body := rec.Body.String()
			for _, s := range tc.wantContains {
				if !strings.Contains(body, s) {
					t.Errorf("response body missing %q", s)
				}
			}
			for _, s := range tc.wantNotContains {
				if strings.Contains(body, s) {
					t.Errorf("response body should not contain %q", s)
				}
			}

			disposition := rec.Header().Get("Content-Disposition")
			if tc.wantDisposition != "" && !strings.Contains(disposition, tc.wantDisposition) {
				t.Errorf("Content-Disposition = %q, want it to contain %q", disposition, tc.wantDisposition)
			}

			contentType := rec.Header().Get("Content-Type")
			if contentType != "application/octet-stream" {
				t.Errorf("Content-Type = %q, want %q", contentType, "application/octet-stream")
			}
		})
	}
}

func TestHandleDownloadRedirect(t *testing.T) {
	tests := []struct {
		name         string
		platform     string
		arch         string
		wantStatus   int
		wantLocation string
	}{
		{
			name:         "Linux",
			platform:     "linux",
			arch:         "amd64",
			wantStatus:   http.StatusFound,
			wantLocation: "scout_linux_amd64",
		},
		{
			name:         "Windows",
			platform:     "windows",
			arch:         "amd64",
			wantStatus:   http.StatusFound,
			wantLocation: ".exe",
		},
		{
			name:       "Invalid",
			platform:   "solaris",
			arch:       "sparc",
			wantStatus: http.StatusBadRequest,
		},
	}

	m := newTestInstallModule()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := "/dispatch/download/" + tc.platform + "/" + tc.arch
			req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
			req.SetPathValue("platform", tc.platform)
			req.SetPathValue("arch", tc.arch)

			rec := httptest.NewRecorder()
			m.handleDownloadRedirect(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}

			if tc.wantStatus == http.StatusFound {
				location := rec.Header().Get("Location")
				if !strings.Contains(location, tc.wantLocation) {
					t.Errorf("Location = %q, want it to contain %q", location, tc.wantLocation)
				}
				if !strings.Contains(location, "github.com/HerbHall/subnetree") {
					t.Errorf("Location = %q, want it to contain GitHub repo URL", location)
				}
			}
		})
	}
}

func TestValidatePlatformArch(t *testing.T) {
	tests := []struct {
		platform string
		arch     string
		wantOK   bool
	}{
		{"linux", "amd64", true},
		{"linux", "arm64", true},
		{"darwin", "amd64", true},
		{"darwin", "arm64", true},
		{"windows", "amd64", true},
		{"windows", "arm64", false},
		{"freebsd", "amd64", false},
		{"linux", "386", false},
	}

	for _, tc := range tests {
		t.Run(tc.platform+"/"+tc.arch, func(t *testing.T) {
			result := validatePlatformArch(tc.platform, tc.arch)
			gotOK := result == ""
			if gotOK != tc.wantOK {
				t.Errorf("validatePlatformArch(%q, %q) = %q, wantOK=%v", tc.platform, tc.arch, result, tc.wantOK)
			}
		})
	}
}

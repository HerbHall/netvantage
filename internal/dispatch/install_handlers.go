package dispatch

import (
	"fmt"
	"net"
	"net/http"

	"github.com/HerbHall/subnetree/internal/version"
	_ "github.com/HerbHall/subnetree/pkg/models" // swagger type reference
)

// validPlatformArch defines valid platform/architecture combinations for Scout binaries.
var validPlatformArch = map[string]map[string]bool{
	"linux":   {"amd64": true, "arm64": true},
	"darwin":  {"amd64": true, "arm64": true},
	"windows": {"amd64": true},
}

// handleInstallScript generates a platform-specific install script for the Scout agent.
//
//	@Summary		Generate install script
//	@Description	Returns a platform-specific install script (bash or PowerShell) for one-click Scout agent deployment.
//	@Tags			dispatch
//	@Produce		application/octet-stream
//	@Security		BearerAuth
//	@Param			platform	path	string	true	"Target OS (linux, darwin, windows)"
//	@Param			arch		path	string	true	"Target architecture (amd64, arm64)"
//	@Param			token		query	string	true	"Enrollment token"
//	@Success		200
//	@Failure		400	{object}	models.APIProblem
//	@Router			/dispatch/install/{platform}/{arch} [get]
func (m *Module) handleInstallScript(w http.ResponseWriter, r *http.Request) {
	platform := r.PathValue("platform")
	arch := r.PathValue("arch")
	token := r.URL.Query().Get("token")

	if token == "" {
		dispatchWriteError(w, http.StatusBadRequest, "query parameter 'token' is required")
		return
	}

	if err := validatePlatformArch(platform, arch); err != "" {
		dispatchWriteError(w, http.StatusBadRequest, err)
		return
	}

	params := m.buildInstallParams(r, platform, arch, token)

	var tmpl = unixInstallTemplate
	filename := fmt.Sprintf("install-scout-%s-%s.sh", platform, arch)
	if platform == "windows" {
		tmpl = windowsInstallTemplate
		filename = fmt.Sprintf("install-scout-%s-%s.ps1", platform, arch)
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))

	if err := tmpl.Execute(w, params); err != nil {
		m.logger.Warn("failed to render install script")
	}
}

// handleDownloadRedirect redirects to the GitHub release binary for the given platform/arch.
//
//	@Summary		Download Scout binary
//	@Description	Redirects to the GitHub release download URL for the Scout binary.
//	@Tags			dispatch
//	@Security		BearerAuth
//	@Param			platform	path	string	true	"Target OS (linux, darwin, windows)"
//	@Param			arch		path	string	true	"Target architecture (amd64, arm64)"
//	@Success		302
//	@Failure		400	{object}	models.APIProblem
//	@Router			/dispatch/download/{platform}/{arch} [get]
func (m *Module) handleDownloadRedirect(w http.ResponseWriter, r *http.Request) {
	platform := r.PathValue("platform")
	arch := r.PathValue("arch")

	if err := validatePlatformArch(platform, arch); err != "" {
		dispatchWriteError(w, http.StatusBadRequest, err)
		return
	}

	downloadURL := buildBinaryURL(platform, arch)
	http.Redirect(w, r, downloadURL, http.StatusFound)
}

// validatePlatformArch checks that the platform/arch combination is valid.
// Returns an empty string on success, or an error message on failure.
func validatePlatformArch(platform, arch string) string {
	archMap, ok := validPlatformArch[platform]
	if !ok {
		return fmt.Sprintf("invalid platform %q; supported: linux, darwin, windows", platform)
	}
	if !archMap[arch] {
		supported := make([]string, 0, len(archMap))
		for a := range archMap {
			supported = append(supported, a)
		}
		return fmt.Sprintf("invalid architecture %q for platform %q; supported: %v", arch, platform, supported)
	}
	return ""
}

// buildBinaryURL constructs the GitHub release download URL for a Scout binary.
func buildBinaryURL(platform, arch string) string {
	ver := version.Short()
	binaryName := fmt.Sprintf("scout_%s_%s", platform, arch)
	if platform == "windows" {
		binaryName += ".exe"
	}

	if ver == "dev" {
		return fmt.Sprintf("https://github.com/HerbHall/subnetree/releases/latest/download/%s", binaryName)
	}
	return fmt.Sprintf("https://github.com/HerbHall/subnetree/releases/download/v%s/%s", ver, binaryName)
}

// buildInstallParams assembles template parameters from the request context and config.
func (m *Module) buildInstallParams(r *http.Request, platform, arch, token string) installScriptParams {
	serverHost := r.Header.Get("X-Forwarded-Host")
	if serverHost == "" {
		serverHost = r.Host
	}

	// Extract bare host (no port) for constructing gRPC address.
	bareHost, _, splitErr := net.SplitHostPort(serverHost)
	if splitErr != nil {
		// Host may not include a port (e.g. behind a reverse proxy).
		bareHost = serverHost
	}

	// Parse gRPC port from config (GRPCAddr is typically ":9090").
	grpcPort := "9090"
	_, cfgPort, splitErr := net.SplitHostPort(m.cfg.GRPCAddr)
	if splitErr == nil && cfgPort != "" {
		grpcPort = cfgPort
	}

	grpcAddr := net.JoinHostPort(bareHost, grpcPort)

	binaryName := "scout"
	if platform == "windows" {
		binaryName = "scout.exe"
	}

	return installScriptParams{
		ServerHost:  serverHost,
		GRPCAddr:    grpcAddr,
		EnrollToken: token,
		BinaryURL:   buildBinaryURL(platform, arch),
		Version:     version.Short(),
		Platform:    platform,
		Arch:        arch,
		BinaryName:  binaryName,
	}
}

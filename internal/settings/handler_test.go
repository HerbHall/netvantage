package settings_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/HerbHall/subnetree/internal/services"
	"github.com/HerbHall/subnetree/internal/settings"
	"github.com/HerbHall/subnetree/internal/testutil"
	"go.uber.org/zap"
)

func setupHandlerEnv(t *testing.T) (*settings.Handler, *http.ServeMux) {
	t.Helper()

	store := testutil.NewStore(t)
	repo, err := services.NewSQLiteSettingsRepository(context.Background(), store)
	if err != nil {
		t.Fatalf("NewSQLiteSettingsRepository: %v", err)
	}

	logger, _ := zap.NewDevelopment()
	handler := settings.NewHandler(repo, logger)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	return handler, mux
}

func doRequest(mux *http.ServeMux, method, path string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func TestHandleListInterfaces(t *testing.T) {
	_, mux := setupHandlerEnv(t)

	w := doRequest(mux, "GET", "/api/v1/settings/interfaces", nil)

	if w.Code != http.StatusOK {
		t.Errorf("ListInterfaces status = %d, want %d", w.Code, http.StatusOK)
	}

	var interfaces []services.NetworkInterface
	if err := json.NewDecoder(w.Body).Decode(&interfaces); err != nil {
		t.Fatalf("Decode response: %v", err)
	}

	// We can't assert specific interfaces exist as they depend on the test environment,
	// but we can verify the response structure is valid JSON array
	t.Logf("Found %d interfaces", len(interfaces))
}

func TestHandleGetScanInterface_NotConfigured(t *testing.T) {
	_, mux := setupHandlerEnv(t)

	w := doRequest(mux, "GET", "/api/v1/settings/scan-interface", nil)

	if w.Code != http.StatusOK {
		t.Errorf("GetScanInterface status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp struct {
		InterfaceName string `json:"interface_name"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Decode response: %v", err)
	}

	if resp.InterfaceName != "" {
		t.Errorf("InterfaceName = %q, want empty string", resp.InterfaceName)
	}
}

func TestHandleSetScanInterface_InvalidInterface(t *testing.T) {
	_, mux := setupHandlerEnv(t)

	w := doRequest(mux, "POST", "/api/v1/settings/scan-interface", map[string]string{
		"interface_name": "nonexistent_interface_xyz",
	})

	if w.Code != http.StatusBadRequest {
		t.Errorf("SetScanInterface with invalid interface status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleSetScanInterface_EmptyInterface(t *testing.T) {
	_, mux := setupHandlerEnv(t)

	// Setting empty interface should succeed (resets to auto-detect)
	w := doRequest(mux, "POST", "/api/v1/settings/scan-interface", map[string]string{
		"interface_name": "",
	})

	if w.Code != http.StatusOK {
		t.Errorf("SetScanInterface with empty interface status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestHandleSetScanInterface_InvalidBody(t *testing.T) {
	_, mux := setupHandlerEnv(t)

	// Send invalid JSON
	req := httptest.NewRequest("POST", "/api/v1/settings/scan-interface", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("SetScanInterface with invalid body status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

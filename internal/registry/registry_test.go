package registry

import (
	"context"
	"errors"
	"testing"

	"github.com/HerbHall/netvantage/pkg/plugin"
	"go.uber.org/zap"
)

// testPlugin is a minimal plugin for testing.
type testPlugin struct {
	info    plugin.PluginInfo
	initErr error
}

func newTestPlugin(name string, deps ...string) *testPlugin {
	return &testPlugin{
		info: plugin.PluginInfo{
			Name:         name,
			Version:      "1.0.0",
			Description:  "test plugin " + name,
			Dependencies: deps,
			APIVersion:   plugin.APIVersionCurrent,
		},
	}
}

func (p *testPlugin) Info() plugin.PluginInfo                              { return p.info }
func (p *testPlugin) Init(_ context.Context, _ plugin.Dependencies) error  { return p.initErr }
func (p *testPlugin) Start(_ context.Context) error                        { return nil }
func (p *testPlugin) Stop(_ context.Context) error                         { return nil }

// testHTTPPlugin implements both Plugin and HTTPProvider.
type testHTTPPlugin struct {
	testPlugin
	routes []plugin.Route
}

func (p *testHTTPPlugin) Routes() []plugin.Route { return p.routes }

func testLogger() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}

func testDeps() func(string) plugin.Dependencies {
	return func(name string) plugin.Dependencies {
		return plugin.Dependencies{
			Logger: testLogger().Named(name),
		}
	}
}

func TestRegister(t *testing.T) {
	reg := New(testLogger())

	p := newTestPlugin("alpha")
	if err := reg.Register(p); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Duplicate registration should fail.
	if err := reg.Register(p); err == nil {
		t.Fatal("Register() expected error for duplicate, got nil")
	}
}

func TestRegisterEmptyName(t *testing.T) {
	reg := New(testLogger())
	p := &testPlugin{info: plugin.PluginInfo{Name: ""}}
	if err := reg.Register(p); err == nil {
		t.Fatal("Register() expected error for empty name, got nil")
	}
}

func TestValidateNoDeps(t *testing.T) {
	reg := New(testLogger())
	reg.Register(newTestPlugin("a"))
	reg.Register(newTestPlugin("b"))

	if err := reg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	all := reg.All()
	if len(all) != 2 {
		t.Fatalf("All() returned %d plugins, want 2", len(all))
	}
}

func TestValidateWithDeps(t *testing.T) {
	reg := New(testLogger())
	reg.Register(newTestPlugin("b", "a")) // b depends on a
	reg.Register(newTestPlugin("a"))

	if err := reg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	// a should come before b in order.
	all := reg.All()
	aIdx, bIdx := -1, -1
	for i, p := range all {
		switch p.Info().Name {
		case "a":
			aIdx = i
		case "b":
			bIdx = i
		}
	}
	if aIdx >= bIdx {
		t.Errorf("expected a (idx %d) before b (idx %d)", aIdx, bIdx)
	}
}

func TestValidateCycleDetection(t *testing.T) {
	reg := New(testLogger())
	reg.Register(newTestPlugin("a", "b"))
	reg.Register(newTestPlugin("b", "a"))

	if err := reg.Validate(); err == nil {
		t.Fatal("Validate() expected cycle error, got nil")
	}
}

func TestValidateMissingRequiredDep(t *testing.T) {
	reg := New(testLogger())
	p := newTestPlugin("a", "missing")
	p.info.Required = true
	reg.Register(p)

	if err := reg.Validate(); err == nil {
		t.Fatal("Validate() expected error for missing required dep, got nil")
	}
}

func TestValidateDisablesOptionalWithMissingDep(t *testing.T) {
	reg := New(testLogger())
	reg.Register(newTestPlugin("a", "missing")) // optional, dep doesn't exist

	if err := reg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if !reg.IsDisabled("a") {
		t.Error("expected plugin 'a' to be disabled")
	}
}

func TestAPIVersionTooOld(t *testing.T) {
	reg := New(testLogger())
	p := newTestPlugin("old")
	p.info.APIVersion = 0 // below APIVersionMin
	p.info.Required = true
	reg.Register(p)

	if err := reg.Validate(); err == nil {
		t.Fatal("Validate() expected error for old API version, got nil")
	}
}

func TestAPIVersionTooNew(t *testing.T) {
	reg := New(testLogger())
	p := newTestPlugin("future")
	p.info.APIVersion = 999 // above APIVersionCurrent
	p.info.Required = true
	reg.Register(p)

	if err := reg.Validate(); err == nil {
		t.Fatal("Validate() expected error for future API version, got nil")
	}
}

func TestInitAll(t *testing.T) {
	reg := New(testLogger())
	reg.Register(newTestPlugin("a"))
	reg.Register(newTestPlugin("b"))
	reg.Validate()

	ctx := context.Background()
	if err := reg.InitAll(ctx, testDeps()); err != nil {
		t.Fatalf("InitAll() error = %v", err)
	}
}

func TestInitAllRequiredFails(t *testing.T) {
	reg := New(testLogger())
	p := newTestPlugin("a")
	p.info.Required = true
	p.initErr = errors.New("init failed")
	reg.Register(p)
	reg.Validate()

	ctx := context.Background()
	if err := reg.InitAll(ctx, testDeps()); err == nil {
		t.Fatal("InitAll() expected error for required plugin failure, got nil")
	}
}

func TestInitAllOptionalDisabledOnFailure(t *testing.T) {
	reg := New(testLogger())
	p := newTestPlugin("a")
	p.initErr = errors.New("init failed")
	reg.Register(p)
	reg.Validate()

	ctx := context.Background()
	if err := reg.InitAll(ctx, testDeps()); err != nil {
		t.Fatalf("InitAll() error = %v", err)
	}
	if !reg.IsDisabled("a") {
		t.Error("expected optional plugin 'a' to be disabled after init failure")
	}
}

func TestStartAllStopAll(t *testing.T) {
	reg := New(testLogger())
	reg.Register(newTestPlugin("a"))
	reg.Validate()

	ctx := context.Background()
	reg.InitAll(ctx, testDeps())

	if err := reg.StartAll(ctx); err != nil {
		t.Fatalf("StartAll() error = %v", err)
	}

	reg.StopAll(ctx) // should not panic
}

func TestGet(t *testing.T) {
	reg := New(testLogger())
	reg.Register(newTestPlugin("a"))
	reg.Validate()

	if _, ok := reg.Get("a"); !ok {
		t.Error("Get('a') returned false, want true")
	}
	if _, ok := reg.Get("nonexistent"); ok {
		t.Error("Get('nonexistent') returned true, want false")
	}
}

func TestAllRoutesHTTPProvider(t *testing.T) {
	reg := New(testLogger())

	hp := &testHTTPPlugin{
		testPlugin: *newTestPlugin("web"),
		routes: []plugin.Route{
			{Method: "GET", Path: "/test"},
		},
	}
	reg.Register(hp)
	reg.Register(newTestPlugin("noroutes")) // no HTTPProvider

	reg.Validate()
	ctx := context.Background()
	reg.InitAll(ctx, testDeps())

	routes := reg.AllRoutes()
	if len(routes) != 1 {
		t.Fatalf("AllRoutes() returned %d plugin route sets, want 1", len(routes))
	}
	if _, ok := routes["web"]; !ok {
		t.Error("AllRoutes() missing 'web' routes")
	}
}

func TestCascadeDisable(t *testing.T) {
	reg := New(testLogger())

	a := newTestPlugin("a")
	a.info.APIVersion = 0 // will be disabled (too old)

	b := newTestPlugin("b", "a") // depends on a

	reg.Register(a)
	reg.Register(b)

	if err := reg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if !reg.IsDisabled("a") {
		t.Error("expected 'a' to be disabled (bad API version)")
	}
	if !reg.IsDisabled("b") {
		t.Error("expected 'b' to be cascade disabled")
	}
}

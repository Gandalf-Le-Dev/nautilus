package plugin

import (
	"context"
	"errors"
	"testing"
)

// testPlugin implements Plugin for testing
type testPlugin struct {
	name         string
	initErr      error
	terminateErr error
	initialized  bool
	terminated   bool
}

func (p *testPlugin) Name() string                         { return p.name }
func (p *testPlugin) Initialize(ctx context.Context) error { p.initialized = true; return p.initErr }
func (p *testPlugin) Terminate(ctx context.Context) error  { p.terminated = true; return p.terminateErr }

func TestPluginRegistration(t *testing.T) {
	r := NewPluginRegistry()

	p := &testPlugin{name: "test-plugin"}
	r.Register(p)

	plugins := r.GetAll()
	if len(plugins) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(plugins))
	}
	if plugins[0].Name() != "test-plugin" {
		t.Fatalf("expected plugin name 'test-plugin', got '%s'", plugins[0].Name())
	}
}

func TestPluginGetByName(t *testing.T) {
	r := NewPluginRegistry()

	p := &testPlugin{name: "my-plugin"}
	r.Register(p)

	found, ok := r.Get("my-plugin")
	if !ok || found == nil {
		t.Fatal("expected to find plugin by name")
	}

	_, ok = r.Get("nonexistent")
	if ok {
		t.Fatal("expected not found for nonexistent plugin")
	}
}

func TestInitializeAll(t *testing.T) {
	r := NewPluginRegistry()

	p1 := &testPlugin{name: "p1"}
	p2 := &testPlugin{name: "p2"}
	r.Register(p1)
	r.Register(p2)

	err := r.InitializeAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p1.initialized || !p2.initialized {
		t.Fatal("expected all plugins to be initialized")
	}
}

func TestInitializeAllReturnsFirstError(t *testing.T) {
	r := NewPluginRegistry()

	expectedErr := errors.New("init failed")
	p1 := &testPlugin{name: "p1", initErr: expectedErr}
	p2 := &testPlugin{name: "p2"}
	r.Register(p1)
	r.Register(p2)

	err := r.InitializeAll(context.Background())
	if err == nil {
		t.Fatal("expected error from InitializeAll")
	}
}

func TestTerminateAllReturnsAggregatedErrors(t *testing.T) {
	r := NewPluginRegistry()

	err1 := errors.New("terminate error 1")
	err2 := errors.New("terminate error 2")
	p1 := &testPlugin{name: "p1", terminateErr: err1}
	p2 := &testPlugin{name: "p2", terminateErr: err2}
	r.Register(p1)
	r.Register(p2)

	errs := r.TerminateAll(context.Background())
	if len(errs) != 2 {
		t.Fatalf("expected 2 termination errors, got %d", len(errs))
	}
	if !p1.terminated || !p2.terminated {
		t.Fatal("expected all plugins to be terminated even on errors")
	}
}

func TestTerminateAllNoErrors(t *testing.T) {
	r := NewPluginRegistry()

	p1 := &testPlugin{name: "p1"}
	p2 := &testPlugin{name: "p2"}
	r.Register(p1)
	r.Register(p2)

	errs := r.TerminateAll(context.Background())
	if len(errs) != 0 {
		t.Fatalf("expected 0 termination errors, got %d", len(errs))
	}
}

// testDatabasePlugin implements both Plugin and DatabasePlugin
type testDatabasePlugin struct {
	testPlugin
}

func (p *testDatabasePlugin) Connect(ctx context.Context) error    { return nil }
func (p *testDatabasePlugin) Disconnect(ctx context.Context) error { return nil }
func (p *testDatabasePlugin) Ping(ctx context.Context) error       { return nil }

func TestGetPluginsByType(t *testing.T) {
	r := NewPluginRegistry()

	dbPlugin := &testDatabasePlugin{testPlugin: testPlugin{name: "db"}}
	regularPlugin := &testPlugin{name: "regular"}
	r.Register(dbPlugin)
	r.Register(regularPlugin)

	dbPlugins := GetPluginsByType[DatabasePlugin](r)
	if len(dbPlugins) != 1 {
		t.Fatalf("expected 1 database plugin, got %d", len(dbPlugins))
	}

	// All plugins should match Plugin interface
	allPlugins := GetPluginsByType[Plugin](r)
	if len(allPlugins) != 2 {
		t.Fatalf("expected 2 plugins matching Plugin interface, got %d", len(allPlugins))
	}
}

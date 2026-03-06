package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/yourusername/tunman/internal/model"
)

func TestMain(m *testing.M) {
	_ = os.Setenv("TUNMAN_LANG", "en")
	os.Exit(m.Run())
}

type fakeStorage struct {
	tunnels []model.Tunnel
	logs    map[string]string
}

func (s *fakeStorage) LoadTunnels() ([]model.Tunnel, error) {
	cloned := make([]model.Tunnel, len(s.tunnels))
	copy(cloned, s.tunnels)
	return cloned, nil
}

func (s *fakeStorage) AddTunnel(tunnel model.Tunnel) error {
	s.tunnels = append(s.tunnels, tunnel)
	return nil
}

func (s *fakeStorage) UpdateTunnel(tunnel model.Tunnel) error {
	for i := range s.tunnels {
		if s.tunnels[i].ID == tunnel.ID {
			s.tunnels[i] = tunnel
			return nil
		}
	}
	return fmt.Errorf("missing tunnel: %s", tunnel.ID)
}

func (s *fakeStorage) DeleteTunnel(id string) error {
	for i := range s.tunnels {
		if s.tunnels[i].ID == id {
			s.tunnels = append(s.tunnels[:i], s.tunnels[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("missing tunnel: %s", id)
}

func (s *fakeStorage) ReadLogs(id string, lines int) (string, error) {
	content := s.logs[id]
	if content == "" {
		return "", nil
	}
	parts := strings.Split(strings.TrimRight(content, "\n"), "\n")
	if lines > 0 && len(parts) > lines {
		parts = parts[len(parts)-lines:]
	}
	return strings.Join(parts, "\n") + "\n", nil
}

func (s *fakeStorage) LogPath(id string) string {
	return "/tmp/" + id + ".log"
}

type fakeController struct {
	startCalls    []string
	stopCalls     []string
	restartCalls  []string
	setupCalls    []string
	teardownCalls []string
	editCalls     []string
	writeCalls    []string
}

func (c *fakeController) CheckCloudflared() error { return nil }
func (c *fakeController) CheckAuth() error        { return nil }
func (c *fakeController) SetupTunnel(tunnel *model.Tunnel) error {
	c.setupCalls = append(c.setupCalls, tunnel.ID)
	tunnel.TunnelID = "cf-123"
	return nil
}
func (c *fakeController) TeardownTunnel(tunnel *model.Tunnel) error {
	c.teardownCalls = append(c.teardownCalls, tunnel.ID)
	return nil
}
func (c *fakeController) Start(tunnel *model.Tunnel) error {
	c.startCalls = append(c.startCalls, tunnel.ID)
	tunnel.Status = model.StatusRunning
	tunnel.SessionName = "tunman-" + tunnel.ID
	tunnel.PID = 1234
	return nil
}
func (c *fakeController) Stop(tunnel *model.Tunnel) error {
	c.stopCalls = append(c.stopCalls, tunnel.ID)
	tunnel.Status = model.StatusStopped
	tunnel.SessionName = ""
	tunnel.PID = 0
	return nil
}
func (c *fakeController) Restart(tunnel *model.Tunnel) error {
	c.restartCalls = append(c.restartCalls, tunnel.ID)
	tunnel.Status = model.StatusRunning
	return nil
}
func (c *fakeController) SyncStatus(tunnels []model.Tunnel) []model.Tunnel { return tunnels }
func (c *fakeController) EditTunnel(tunnel *model.Tunnel) error {
	c.editCalls = append(c.editCalls, tunnel.ID)
	tunnel.Description = "edited"
	return nil
}
func (c *fakeController) WriteConfigYML(tunnel *model.Tunnel) error {
	c.writeCalls = append(c.writeCalls, tunnel.ID)
	return nil
}

func TestCLIListSupportsFilters(t *testing.T) {
	now := model.FlexTime(time.Now())
	storage := &fakeStorage{tunnels: []model.Tunnel{
		{ID: "custom-1", Name: "api", Category: model.CategoryCustom, Status: model.StatusRunning, Port: 3000, BaseDomain: "example.com", Prefix: "api", CreatedAt: now, UpdatedAt: now},
		{ID: "ephemeral-1", Name: "quick-8080", Category: model.CategoryEphemeral, Status: model.StatusStopped, Port: 8080, CreatedAt: now, UpdatedAt: now},
	}}
	controller := &fakeController{}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	app := &cliApp{stdout: stdout, stderr: stderr, storage: storage, controller: controller, version: "test"}

	code := app.Run([]string{"list", "--status", "running"})
	if code != 0 {
		t.Fatalf("Run() code = %d, stderr = %q", code, stderr.String())
	}
	if got := stdout.String(); !strings.Contains(got, "custom-1") || strings.Contains(got, "ephemeral-1") {
		t.Fatalf("unexpected filtered list output: %q", got)
	}
	if !strings.Contains(stdout.String(), "total=1") {
		t.Fatalf("expected filtered summary, got %q", stdout.String())
	}
}

func TestCLIStatusShowsTunnelDetails(t *testing.T) {
	now := model.FlexTime(time.Now())
	storage := &fakeStorage{tunnels: []model.Tunnel{{
		ID:          "custom-1",
		Name:        "api",
		Category:    model.CategoryCustom,
		Status:      model.StatusRunning,
		Port:        3000,
		BaseDomain:  "example.com",
		Prefix:      "api",
		TunnelID:    "cf-123",
		SessionName: "tunman-custom-1",
		PID:         4321,
		CreatedAt:   now,
		UpdatedAt:   now,
	}}}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	app := &cliApp{stdout: stdout, stderr: stderr, storage: storage, controller: &fakeController{}, version: "test"}

	code := app.Run([]string{"status", "custom-1"})
	if code != 0 {
		t.Fatalf("Run() code = %d, stderr = %q", code, stderr.String())
	}
	got := stdout.String()
	for _, want := range []string{"ID: custom-1", "Name: api", "Base domain: example.com", "Tunnel ID: cf-123", "Session: tunman-custom-1", "PID: 4321"} {
		if !strings.Contains(got, want) {
			t.Fatalf("status output missing %q in %q", want, got)
		}
	}
}

func TestCLICreateEphemeralTunnel(t *testing.T) {
	storage := &fakeStorage{}
	controller := &fakeController{}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	app := &cliApp{stdout: stdout, stderr: stderr, storage: storage, controller: controller, version: "test"}

	code := app.Run([]string{"create", "ephemeral", "--port", "8080", "--desc", "quick"})
	if code != 0 {
		t.Fatalf("Run() code = %d, stderr = %q", code, stderr.String())
	}
	if len(storage.tunnels) != 1 {
		t.Fatalf("expected 1 tunnel, got %d", len(storage.tunnels))
	}
	created := storage.tunnels[0]
	if created.Category != model.CategoryEphemeral || created.Port != 8080 || created.Description != "quick" {
		t.Fatalf("unexpected created tunnel: %+v", created)
	}
	if len(controller.startCalls) != 0 {
		t.Fatalf("ephemeral create should not auto-start, got start calls %v", controller.startCalls)
	}
	if !strings.Contains(stdout.String(), "Created:") {
		t.Fatalf("expected created output, got %q", stdout.String())
	}
}

func TestCLITempAliasCreatesEphemeralTunnel(t *testing.T) {
	storage := &fakeStorage{}
	controller := &fakeController{}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	app := &cliApp{stdout: stdout, stderr: stderr, storage: storage, controller: controller, version: "test"}

	code := app.Run([]string{"temp", "9090", "--desc", "demo"})
	if code != 0 {
		t.Fatalf("Run() code = %d, stderr = %q", code, stderr.String())
	}
	if len(storage.tunnels) != 1 || storage.tunnels[0].Category != model.CategoryEphemeral || storage.tunnels[0].Port != 9090 {
		t.Fatalf("unexpected temp tunnel: %+v", storage.tunnels)
	}
}

func TestCLIReturnsUsageExitCode(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	app := &cliApp{stdout: stdout, stderr: stderr, storage: &fakeStorage{}, controller: &fakeController{}, version: "test"}

	code := app.Run([]string{"start"})
	if code != 2 {
		t.Fatalf("Run() code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "Usage: tunman start <id>") {
		t.Fatalf("expected usage message, got %q", stderr.String())
	}
}

func TestCLILogsReadsRequestedTail(t *testing.T) {
	now := model.FlexTime(time.Now())
	storage := &fakeStorage{
		tunnels: []model.Tunnel{{ID: "custom-1", Name: "api", Category: model.CategoryCustom, Status: model.StatusRunning, Port: 3000, CreatedAt: now, UpdatedAt: now}},
		logs:    map[string]string{"custom-1": "line1\nline2\nline3\n"},
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	app := &cliApp{stdout: stdout, stderr: stderr, storage: storage, controller: &fakeController{}, version: "test"}

	code := app.Run([]string{"logs", "--lines", "2", "custom-1"})
	if code != 0 {
		t.Fatalf("Run() code = %d, stderr = %q", code, stderr.String())
	}
	if got := stdout.String(); got != "line2\nline3\n" {
		t.Fatalf("logs output = %q, want %q", got, "line2\\nline3\\n")
	}
}

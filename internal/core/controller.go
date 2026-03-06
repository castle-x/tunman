package core

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/yourusername/tunman/internal/model"
)

// Controller 隧道控制器
type Controller struct {
	storage *Storage
}

// NewController 创建控制器
func NewController(storage *Storage) *Controller {
	return &Controller{storage: storage}
}

// ---------------------------------------------------------------------------
// cloudflared 环境检查
// ---------------------------------------------------------------------------

// CheckCloudflared 检查 cloudflared 是否安装
func (c *Controller) CheckCloudflared() error {
	if _, err := exec.LookPath("cloudflared"); err != nil {
		return fmt.Errorf("cloudflared not found in PATH; please install it first")
	}
	return nil
}

// CheckAuth 检查 cloudflared 是否已登录 (cert.pem 存在)
func (c *Controller) CheckAuth() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home dir: %w", err)
	}
	certPath := filepath.Join(home, ".cloudflared", "cert.pem")
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		return fmt.Errorf("not logged in; run 'cloudflared tunnel login' first")
	}
	return nil
}

// ---------------------------------------------------------------------------
// 隧道生命周期：Setup / Teardown
// ---------------------------------------------------------------------------

// SetupTunnel 创建 cloudflared tunnel + 配置 DNS + 生成 config
// 仅 custom / testing 需要；ephemeral 无需 setup。
func (c *Controller) SetupTunnel(tunnel *model.Tunnel) error {
	if tunnel.Category == model.CategoryEphemeral {
		return nil
	}

	if err := c.CheckCloudflared(); err != nil {
		return err
	}
	if err := c.CheckAuth(); err != nil {
		return err
	}

	cfName := tunnel.CloudflaredName()

	// 1. cloudflared tunnel create <name>
	out, err := exec.Command("cloudflared", "tunnel", "create", cfName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("tunnel create failed: %s — %w", strings.TrimSpace(string(out)), err)
	}
	tunnelID := parseTunnelID(string(out))
	if tunnelID == "" {
		return fmt.Errorf("could not parse tunnel ID from cloudflared output")
	}
	tunnel.TunnelID = tunnelID

	// 2. cloudflared tunnel route dns <name> <full-domain>
	fullDomain := tunnel.FullDomain()
	if fullDomain == "" {
		return fmt.Errorf("full domain is empty; cannot route DNS")
	}
	out, err = exec.Command("cloudflared", "tunnel", "route", "dns", "--overwrite-dns", cfName, fullDomain).CombinedOutput()
	if err != nil {
		return fmt.Errorf("dns route failed: %s — %w", strings.TrimSpace(string(out)), err)
	}

	// 3. 生成 config.yml
	if err := c.WriteConfigYML(tunnel); err != nil {
		return fmt.Errorf("write config failed: %w", err)
	}

	return nil
}

// TeardownTunnel 删除 cloudflared tunnel 及其配置
func (c *Controller) TeardownTunnel(tunnel *model.Tunnel) error {
	if tunnel.Category == model.CategoryEphemeral || tunnel.TunnelID == "" {
		return nil
	}

	cfName := tunnel.CloudflaredName()

	// 尝试清理 DNS 路由（忽略错误，可能已清理）
	if fd := tunnel.FullDomain(); fd != "" {
		exec.Command("cloudflared", "tunnel", "route", "dns", "--remove", cfName, fd).Run()
	}

	// 删除 tunnel
	out, err := exec.Command("cloudflared", "tunnel", "delete", cfName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("tunnel delete failed: %s — %w", strings.TrimSpace(string(out)), err)
	}

	// 清理本地 config
	cfgDir := c.configDir(tunnel.ID)
	os.RemoveAll(cfgDir)

	return nil
}

// ---------------------------------------------------------------------------
// 隧道运行控制：Start / Stop / Restart
// ---------------------------------------------------------------------------

// Start 启动隧道
func (c *Controller) Start(tunnel *model.Tunnel) error {
	if tunnel.Status == model.StatusRunning {
		return fmt.Errorf("tunnel already running")
	}

	sessionName := fmt.Sprintf("tunman-%s", tunnel.ID)
	logPath := c.storage.LogPath(tunnel.ID)

	var cmdStr string
	switch tunnel.Category {
	case model.CategoryEphemeral:
		cmdStr = fmt.Sprintf("cloudflared tunnel --url http://localhost:%d 2>&1 | tee -a %s",
			tunnel.Port, logPath)
	default:
		cfgPath := c.configPath(tunnel.ID)
		cfName := tunnel.CloudflaredName()
		cmdStr = fmt.Sprintf("cloudflared tunnel --config %s run %s 2>&1 | tee -a %s",
			cfgPath, cfName, logPath)
	}

	cmd := exec.Command("screen", "-dmS", sessionName, "bash", "-c", cmdStr)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start screen session: %w", err)
	}

	time.Sleep(500 * time.Millisecond)
	pid, err := c.getScreenPID(sessionName)
	if err == nil {
		tunnel.PID = pid
	}

	tunnel.Status = model.StatusRunning
	tunnel.SessionName = sessionName
	return nil
}

// Stop 停止隧道
func (c *Controller) Stop(tunnel *model.Tunnel) error {
	if tunnel.Status != model.StatusRunning {
		return fmt.Errorf("tunnel not running")
	}

	sessionName := fmt.Sprintf("tunman-%s", tunnel.ID)
	exec.Command("screen", "-S", sessionName, "-X", "quit").Run()

	tunnel.Status = model.StatusStopped
	tunnel.PID = 0
	tunnel.SessionName = ""
	return nil
}

// Restart 重启隧道
func (c *Controller) Restart(tunnel *model.Tunnel) error {
	c.Stop(tunnel)
	time.Sleep(500 * time.Millisecond)
	return c.Start(tunnel)
}

// SyncStatus 同步隧道状态
func (c *Controller) SyncStatus(tunnels []model.Tunnel) []model.Tunnel {
	for i := range tunnels {
		sessionName := fmt.Sprintf("tunman-%s", tunnels[i].ID)
		if c.screenExists(sessionName) {
			tunnels[i].Status = model.StatusRunning
			tunnels[i].SessionName = sessionName
			pid, _ := c.getScreenPID(sessionName)
			tunnels[i].PID = pid
		} else {
			tunnels[i].Status = model.StatusStopped
			tunnels[i].PID = 0
			tunnels[i].SessionName = ""
		}
	}
	return tunnels
}

// GetLogs 获取日志
func (c *Controller) GetLogs(tunnel *model.Tunnel, tail bool) (string, error) {
	logPath := c.storage.LogPath(tunnel.ID)

	if tail {
		return "", nil
	}

	cmd := exec.Command("tail", "-n", "100", logPath)
	output, err := cmd.Output()
	if err != nil {
		if os.IsNotExist(err) {
			return "No logs yet", nil
		}
		return "", err
	}
	return string(output), nil
}

// CleanupEphemeral 清理所有临时隧道
func (c *Controller) CleanupEphemeral() {
	sessions := c.listScreenSessions()
	for _, session := range sessions {
		if strings.HasPrefix(session, "tunman-ephemeral-") {
			exec.Command("screen", "-S", session, "-X", "quit").Run()
		}
	}
}

// ---------------------------------------------------------------------------
// 编辑器相关
// ---------------------------------------------------------------------------

// OpenEditor 用外部编辑器编辑文件
func (c *Controller) OpenEditor(path string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}
	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// EditTunnel 编辑隧道配置
func (c *Controller) EditTunnel(tunnel *model.Tunnel) error {
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, fmt.Sprintf("tunman-%s.json", tunnel.ID))

	data, _ := json.MarshalIndent(tunnel, "", "  ")
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return err
	}

	if err := c.OpenEditor(tmpFile); err != nil {
		return err
	}

	newData, err := os.ReadFile(tmpFile)
	if err != nil {
		return err
	}

	var updated model.Tunnel
	if err := json.Unmarshal(newData, &updated); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	*tunnel = updated
	return c.storage.UpdateTunnel(*tunnel)
}

// ---------------------------------------------------------------------------
// config.yml 管理
// ---------------------------------------------------------------------------

func (c *Controller) configDir(tunnelRecordID string) string {
	return filepath.Join(c.storage.BaseDir, "configs", tunnelRecordID)
}

func (c *Controller) configPath(tunnelRecordID string) string {
	return filepath.Join(c.configDir(tunnelRecordID), "config.yml")
}

// WriteConfigYML 生成/重写 cloudflared config.yml
func (c *Controller) WriteConfigYML(tunnel *model.Tunnel) error {
	dir := c.configDir(tunnel.ID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	home, _ := os.UserHomeDir()
	credsFile := filepath.Join(home, ".cloudflared", tunnel.TunnelID+".json")

	cfg := fmt.Sprintf(`tunnel: %s
credentials-file: %s

ingress:
  - hostname: %s
    service: http://localhost:%d
  - service: http_status:404
`, tunnel.TunnelID, credsFile, tunnel.FullDomain(), tunnel.Port)

	return os.WriteFile(c.configPath(tunnel.ID), []byte(cfg), 0644)
}

// ---------------------------------------------------------------------------
// screen 辅助
// ---------------------------------------------------------------------------

func (c *Controller) screenExists(name string) bool {
	cmd := exec.Command("screen", "-ls", name)
	output, _ := cmd.Output()
	return strings.Contains(string(output), name)
}

func (c *Controller) getScreenPID(name string) (int, error) {
	cmd := exec.Command("screen", "-ls")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	re := regexp.MustCompile(`(\d+)\.\Q` + name + `\E`)
	matches := re.FindStringSubmatch(string(output))
	if len(matches) > 1 {
		return strconv.Atoi(matches[1])
	}
	return 0, fmt.Errorf("PID not found")
}

func (c *Controller) listScreenSessions() []string {
	cmd := exec.Command("screen", "-ls")
	output, _ := cmd.Output()

	var sessions []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) > 0 && strings.Contains(parts[0], ".") {
			session := strings.Split(parts[0], ".")[1]
			sessions = append(sessions, session)
		}
	}
	return sessions
}

// ---------------------------------------------------------------------------
// 解析工具
// ---------------------------------------------------------------------------

// parseTunnelID 从 cloudflared tunnel create 输出中提取 UUID
// 典型输出: "Created tunnel tunman-xxx with id 8c9b5c5f-1234-5678-9abc-def012345678"
func parseTunnelID(output string) string {
	re := regexp.MustCompile(`with id ([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})`)
	matches := re.FindStringSubmatch(output)
	if len(matches) > 1 {
		return matches[1]
	}

	uuidRe := regexp.MustCompile(`([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})`)
	matches = uuidRe.FindStringSubmatch(output)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yourusername/tunman/internal/model"
)

// Storage 数据存储
type Storage struct {
	BaseDir string
}

// NewStorage 创建存储实例
func NewStorage() *Storage {
	home, _ := os.UserHomeDir()
	return &Storage{
		BaseDir: filepath.Join(home, ".tunman"),
	}
}

// ensureDir 确保目录存在
func (s *Storage) ensureDir() error {
	return os.MkdirAll(s.BaseDir, 0755)
}

// LoadConfig 加载配置
func (s *Storage) LoadConfig() (*model.Config, error) {
	path := filepath.Join(s.BaseDir, "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return model.DefaultConfig(), nil
		}
		return nil, err
	}

	var cfg model.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// SaveConfig 保存配置
func (s *Storage) SaveConfig(cfg *model.Config) error {
	if err := s.ensureDir(); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(s.BaseDir, "config.json"), data, 0644)
}

// LoadTunnels 加载所有隧道
func (s *Storage) LoadTunnels() ([]model.Tunnel, error) {
	path := filepath.Join(s.BaseDir, "tunnels.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []model.Tunnel{}, nil
		}
		return nil, err
	}

	var wrapper struct {
		Tunnels []model.Tunnel `json:"tunnels"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, err
	}

	return wrapper.Tunnels, nil
}

// SaveTunnels 保存隧道列表
func (s *Storage) SaveTunnels(tunnels []model.Tunnel) error {
	if err := s.ensureDir(); err != nil {
		return err
	}

	wrapper := struct {
		Tunnels []model.Tunnel `json:"tunnels"`
	}{
		Tunnels: tunnels,
	}

	data, err := json.MarshalIndent(wrapper, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(s.BaseDir, "tunnels.json"), data, 0644)
}

// GetTunnel 获取单个隧道
func (s *Storage) GetTunnel(id string) (*model.Tunnel, error) {
	tunnels, err := s.LoadTunnels()
	if err != nil {
		return nil, err
	}

	for _, t := range tunnels {
		if t.ID == id {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("tunnel not found: %s", id)
}

// AddTunnel 添加隧道
func (s *Storage) AddTunnel(tunnel model.Tunnel) error {
	tunnels, err := s.LoadTunnels()
	if err != nil {
		return err
	}

	// 检查 ID 是否已存在
	for _, t := range tunnels {
		if t.ID == tunnel.ID {
			return fmt.Errorf("tunnel already exists: %s", tunnel.ID)
		}
	}

	now := model.FlexTime(time.Now())
	tunnel.CreatedAt = now
	tunnel.UpdatedAt = now

	tunnels = append(tunnels, tunnel)
	return s.SaveTunnels(tunnels)
}

// UpdateTunnel 更新隧道
func (s *Storage) UpdateTunnel(tunnel model.Tunnel) error {
	tunnels, err := s.LoadTunnels()
	if err != nil {
		return err
	}

	found := false
	for i, t := range tunnels {
		if t.ID == tunnel.ID {
			tunnel.UpdatedAt = model.FlexTime(time.Now())
			tunnels[i] = tunnel
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("tunnel not found: %s", tunnel.ID)
	}

	return s.SaveTunnels(tunnels)
}

// DeleteTunnel 删除隧道
func (s *Storage) DeleteTunnel(id string) error {
	tunnels, err := s.LoadTunnels()
	if err != nil {
		return err
	}

	var filtered []model.Tunnel
	found := false
	for _, t := range tunnels {
		if t.ID != id {
			filtered = append(filtered, t)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("tunnel not found: %s", id)
	}

	return s.SaveTunnels(filtered)
}

// LogPath 获取日志文件路径
func (s *Storage) LogPath(id string) string {
	logsDir := filepath.Join(s.BaseDir, "logs")
	os.MkdirAll(logsDir, 0755)
	return filepath.Join(logsDir, fmt.Sprintf("%s.log", id))
}

// ReadLogs 读取日志内容
func (s *Storage) ReadLogs(id string, lines int) (string, error) {
	path := s.LogPath(id)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	content := strings.ReplaceAll(string(data), "\r\n", "\n")
	if lines <= 0 {
		return content, nil
	}

	trimmed := strings.TrimRight(content, "\n")
	if trimmed == "" {
		return "", nil
	}

	parts := strings.Split(trimmed, "\n")
	if lines < len(parts) {
		parts = parts[len(parts)-lines:]
	}

	return strings.Join(parts, "\n") + "\n", nil
}

package model

import (
	"crypto/rand"
	"fmt"
	"strings"
)

// Category 隧道分类
type Category string

const (
	CategoryCustom    Category = "custom"
	CategoryTesting   Category = "testing"
	CategoryEphemeral Category = "ephemeral"
)

func (c Category) Icon() string {
	switch c {
	case CategoryCustom:
		return "🌐"
	case CategoryTesting:
		return "🧪"
	case CategoryEphemeral:
		return "⏱️"
	default:
		return "❓"
	}
}

func (c Category) String() string {
	return string(c)
}

func (c Category) DisplayName() string {
	switch c {
	case CategoryCustom:
		return "Custom"
	case CategoryTesting:
		return "Testing"
	case CategoryEphemeral:
		return "Ephemeral"
	default:
		return "Unknown"
	}
}

func ParseCategory(value string) (Category, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "custom":
		return CategoryCustom, nil
	case "testing", "test":
		return CategoryTesting, nil
	case "ephemeral", "quick", "temp":
		return CategoryEphemeral, nil
	default:
		return "", fmt.Errorf("unknown category: %s", value)
	}
}

// Status 运行状态
type Status string

const (
	StatusStopped Status = "stopped"
	StatusRunning Status = "running"
	StatusError   Status = "error"
)

func (s Status) String() string {
	return string(s)
}

func (s Status) Icon() string {
	switch s {
	case StatusRunning:
		return "●"
	case StatusStopped:
		return "○"
	case StatusError:
		return "✗"
	default:
		return "?"
	}
}

func ParseStatus(value string) (Status, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "running", "run":
		return StatusRunning, nil
	case "stopped", "stop":
		return StatusStopped, nil
	case "error", "err":
		return StatusError, nil
	default:
		return "", fmt.Errorf("unknown status: %s", value)
	}
}

// Tunnel 隧道定义
type Tunnel struct {
	ID          string   `json:"id"`
	Category    Category `json:"category"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Port        int      `json:"local_port"`

	// custom + testing: 用户提供根域名，prefix 为三级前缀
	BaseDomain string `json:"base_domain,omitempty"`
	Prefix     string `json:"prefix,omitempty"`

	// controller 自动管理，用户不直接填写
	TunnelID string `json:"tunnel_id,omitempty"`

	// 运行时状态（不持久化）
	Status      Status `json:"-"`
	PID         int    `json:"-"`
	SessionName string `json:"-"`

	CreatedAt FlexTime `json:"created_at"`
	UpdatedAt FlexTime `json:"updated_at"`
}

// Config 工具配置
type Config struct {
	Version         string `json:"version"`
	AutoRefresh     bool   `json:"auto_refresh"`
	RefreshInterval int    `json:"refresh_interval"`
	Editor          string `json:"editor"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Version:         "2.0",
		AutoRefresh:     true,
		RefreshInterval: 5,
		Editor:          "vim",
	}
}

// FullDomain 计算完整域名: prefix.baseDomain
func (t *Tunnel) FullDomain() string {
	if t.Prefix == "" || t.BaseDomain == "" {
		return ""
	}
	return fmt.Sprintf("%s.%s", t.Prefix, t.BaseDomain)
}

// DisplayURL 返回显示用的访问地址
func (t *Tunnel) DisplayURL() string {
	if fd := t.FullDomain(); fd != "" {
		return fmt.Sprintf("https://%s", fd)
	}
	return fmt.Sprintf(":%d", t.Port)
}

// IsEphemeral 是否为临时隧道
func (t *Tunnel) IsEphemeral() bool {
	return t.Category == CategoryEphemeral
}

// GenerateTestingPrefix 为 testing 类型生成随机短 hex 前缀
func GenerateTestingPrefix() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// CloudflaredName 用于 cloudflared tunnel create 的名称（全局唯一）
func (t *Tunnel) CloudflaredName() string {
	return fmt.Sprintf("tunman-%s", t.ID)
}

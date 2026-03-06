package core

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/yourusername/tunman/internal/model"
)

type CreateTunnelInput struct {
	Category    model.Category
	BaseDomain  string
	Prefix      string
	Port        int
	Description string
}

func (in CreateTunnelInput) Validate() error {
	if in.Port <= 0 || in.Port > 65535 {
		return fmt.Errorf("port must be 1-65535")
	}

	switch in.Category {
	case model.CategoryCustom:
		if strings.TrimSpace(in.BaseDomain) == "" {
			return fmt.Errorf("base domain is required")
		}
		if strings.TrimSpace(in.Prefix) == "" {
			return fmt.Errorf("subdomain prefix is required for custom")
		}
	case model.CategoryTesting:
		if strings.TrimSpace(in.BaseDomain) == "" {
			return fmt.Errorf("base domain is required")
		}
	case model.CategoryEphemeral:
	default:
		return fmt.Errorf("unsupported category: %s", in.Category)
	}

	return nil
}

func (in CreateTunnelInput) Build() (model.Tunnel, error) {
	if err := in.Validate(); err != nil {
		return model.Tunnel{}, err
	}

	tunnel := model.Tunnel{
		Category:    in.Category,
		Description: strings.TrimSpace(in.Description),
		Port:        in.Port,
		Status:      model.StatusStopped,
	}

	switch in.Category {
	case model.CategoryCustom:
		tunnel.BaseDomain = strings.TrimSpace(in.BaseDomain)
		tunnel.Prefix = strings.TrimSpace(in.Prefix)
		tunnel.Name = tunnel.Prefix
		tunnel.ID = buildTunnelRecordID(tunnel.Name)
	case model.CategoryTesting:
		tunnel.BaseDomain = strings.TrimSpace(in.BaseDomain)
		tunnel.Prefix = model.GenerateTestingPrefix()
		tunnel.Name = fmt.Sprintf("test-%s", tunnel.Prefix)
		tunnel.ID = buildTunnelRecordID(tunnel.Name)
	case model.CategoryEphemeral:
		tunnel.Name = fmt.Sprintf("quick-%d", in.Port)
		tunnel.ID = buildTunnelRecordID(tunnel.Name)
	}

	return tunnel, nil
}

func CreateManagedTunnel(storage *Storage, controller *Controller, input CreateTunnelInput) (model.Tunnel, error) {
	tunnel, err := input.Build()
	if err != nil {
		return model.Tunnel{}, err
	}

	if tunnel.Category != model.CategoryEphemeral {
		if err := controller.CheckCloudflared(); err != nil {
			return model.Tunnel{}, err
		}
		if err := controller.CheckAuth(); err != nil {
			return model.Tunnel{}, err
		}
		if err := controller.SetupTunnel(&tunnel); err != nil {
			return model.Tunnel{}, err
		}
	}

	if err := storage.AddTunnel(tunnel); err != nil {
		return model.Tunnel{}, err
	}

	if tunnel.Category == model.CategoryEphemeral {
		return tunnel, nil
	}

	if err := controller.Start(&tunnel); err != nil {
		return tunnel, fmt.Errorf("created tunnel but failed to start: %w", err)
	}
	if err := storage.UpdateTunnel(tunnel); err != nil {
		return tunnel, err
	}

	return tunnel, nil
}

func LoadSyncedTunnels(storage *Storage, controller *Controller) ([]model.Tunnel, error) {
	tunnels, err := storage.LoadTunnels()
	if err != nil {
		return nil, err
	}
	return controller.SyncStatus(tunnels), nil
}

func StartManagedTunnel(storage *Storage, controller *Controller, id string) (model.Tunnel, error) {
	tunnel, err := loadTunnel(storage, controller, id)
	if err != nil {
		return model.Tunnel{}, err
	}
	if err := ensureRunnable(controller, &tunnel); err != nil {
		return model.Tunnel{}, err
	}
	if err := controller.Start(&tunnel); err != nil {
		return model.Tunnel{}, err
	}
	if err := storage.UpdateTunnel(tunnel); err != nil {
		return model.Tunnel{}, err
	}
	return tunnel, nil
}

func StopManagedTunnel(storage *Storage, controller *Controller, id string) (model.Tunnel, error) {
	tunnel, err := loadTunnel(storage, controller, id)
	if err != nil {
		return model.Tunnel{}, err
	}
	if err := controller.Stop(&tunnel); err != nil {
		return model.Tunnel{}, err
	}
	if err := storage.UpdateTunnel(tunnel); err != nil {
		return model.Tunnel{}, err
	}
	return tunnel, nil
}

func RestartManagedTunnel(storage *Storage, controller *Controller, id string) (model.Tunnel, error) {
	tunnel, err := loadTunnel(storage, controller, id)
	if err != nil {
		return model.Tunnel{}, err
	}
	if err := ensureRunnable(controller, &tunnel); err != nil {
		return model.Tunnel{}, err
	}
	if err := controller.Restart(&tunnel); err != nil {
		return model.Tunnel{}, err
	}
	if err := storage.UpdateTunnel(tunnel); err != nil {
		return model.Tunnel{}, err
	}
	return tunnel, nil
}

func DeleteManagedTunnel(storage *Storage, controller *Controller, id string) (model.Tunnel, error) {
	tunnel, err := loadTunnel(storage, controller, id)
	if err != nil {
		return model.Tunnel{}, err
	}
	if tunnel.Status == model.StatusRunning {
		if err := controller.Stop(&tunnel); err != nil {
			return model.Tunnel{}, err
		}
	}
	if err := controller.TeardownTunnel(&tunnel); err != nil {
		return model.Tunnel{}, err
	}
	if err := storage.DeleteTunnel(tunnel.ID); err != nil {
		return model.Tunnel{}, err
	}
	return tunnel, nil
}

func loadTunnel(storage *Storage, controller *Controller, id string) (model.Tunnel, error) {
	tunnels, err := LoadSyncedTunnels(storage, controller)
	if err != nil {
		return model.Tunnel{}, err
	}
	for _, tunnel := range tunnels {
		if tunnel.ID == id {
			return tunnel, nil
		}
	}
	return model.Tunnel{}, fmt.Errorf("tunnel not found: %s", id)
}

func ensureRunnable(controller *Controller, tunnel *model.Tunnel) error {
	if tunnel.Category == model.CategoryEphemeral {
		return nil
	}
	if strings.TrimSpace(tunnel.TunnelID) == "" {
		return fmt.Errorf("tunnel %s is missing tunnel_id", tunnel.ID)
	}
	return controller.WriteConfigYML(tunnel)
}

func buildTunnelRecordID(name string) string {
	base := strings.ToLower(strings.TrimSpace(name))
	base = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(base, "-")
	base = strings.Trim(base, "-")
	if base == "" {
		base = "tunnel"
	}
	return fmt.Sprintf("%s-%d", base, time.Now().Unix())
}

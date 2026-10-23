package xray

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"awewe-panel/internal/models"
)

type Manager struct {
	bin      string
	confPath string
	mu       sync.Mutex
	cmd      *exec.Cmd
}

func New(bin, confPath string) *Manager {
	return &Manager{bin: bin, confPath: confPath}
}

func (m *Manager) buildConfig(nodes []models.Node) map[string]interface{} {
	inbounds := []map[string]interface{}{}
	for _, n := range nodes {
		// Пропускаем автовыбор ноды - они не создают реальных inbound
		if n.Type == "auto" {
			continue
		}

		inbound := map[string]interface{}{
			"listen":   "0.0.0.0",
			"port":     n.Port,
			"protocol": "vless",
			"settings": map[string]interface{}{
				"clients":    []map[string]interface{}{{"id": n.UUID, "level": 0}},
				"decryption": "none",
			},
			"streamSettings": map[string]interface{}{
				"network":  "xhttp",
				"security": "none",
				"xhttpSettings": map[string]interface{}{
					"path":             n.Path,
					"mode":             "auto",
					"uplinkHTTPMethod": n.UplinkHTTPMethod,
				},
			},
			"tag": fmt.Sprintf("in-%d", n.ID),
		}

		// Добавляем дополнительные параметры если есть
		if n.Encryption != "" && n.Encryption != "none" {
			inbound["streamSettings"].(map[string]interface{})["security"] = n.Encryption
		}
		if n.Flow != "" && n.Flow != "none" {
			inbound["settings"].(map[string]interface{})["flow"] = n.Flow
		}
		if n.Transport != "" && n.Transport != "TCP" {
			inbound["streamSettings"].(map[string]interface{})["network"] = n.Transport
		}

		inbounds = append(inbounds, inbound)
	}

	return map[string]interface{}{
		"log":      map[string]interface{}{"loglevel": "warning"},
		"inbounds": inbounds,
		"outbounds": []map[string]interface{}{
			{"protocol": "freedom", "tag": "direct"},
		},
	}
}

func (m *Manager) writeConfig(nodes []models.Node) error {
	cfg := m.buildConfig(nodes)
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.confPath, b, 0644)
}

func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cmd != nil && m.cmd.Process != nil {
		_ = m.cmd.Process.Kill()
		_ = m.cmd.Wait()
		m.cmd = nil
	}
}

func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cmd := exec.Command(m.bin, "run", "-c", m.confPath)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start xray: %w", err)
	}
	m.cmd = cmd
	go func() { _ = cmd.Wait() }()
	return nil
}

func (m *Manager) Restart(nodes []models.Node) error {
	m.Stop()
	if err := m.writeConfig(nodes); err != nil {
		return err
	}
	return m.Start()
}

func (m *Manager) Running() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cmd != nil && m.cmd.Process != nil
}

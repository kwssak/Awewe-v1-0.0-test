package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	Port       int    `json:"port"`
	WebPath    string `json:"web_path"`
	APIToken   string `json:"api_token"`
	XrayConfig string `json:"xray_config"`
	XrayBin    string `json:"xray_bin"`
	DBPath     string `json:"db"`
	WebDir     string `json:"web_dir"`
	SubBase    string `json:"sub_base"`
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	c := &Config{}
	if err := json.Unmarshal(b, c); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if c.SubBase == "" {
		c.SubBase = "/sub"
	}
	return c, nil
}

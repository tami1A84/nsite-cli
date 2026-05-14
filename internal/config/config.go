package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"

	"github.com/nbd-wtf/go-nostr/nip19"
)

type Relay struct {
	Read   bool `json:"read"`
	Write  bool `json:"write"`
	Search bool `json:"search"`
}

type Blossom struct {
	Servers []string `json:"servers"`
}

type Nsite struct {
	// Host is optional. When set, publish/inspect can print canonical NIP-5A URLs.
	// Example: "nsite.example.com".
	Host string `json:"host,omitempty"`
}

type Config struct {
	Relays         map[string]Relay `json:"relays"`
	PrivateKey     string           `json:"privatekey"`
	Blossom        Blossom          `json:"blossom"`
	BlossomServers []string         `json:"blossomServers,omitempty"` // legacy/shortcut form
	Nsite          Nsite            `json:"nsite,omitempty"`
}

func ConfigDir() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".config", "nsite-cli"), nil
	default:
		base, err := os.UserConfigDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(base, "nsite-cli"), nil
	}
}

func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func Load() (*Config, error) {
	fp, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(fp)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", fp, err)
	}
	var cfg Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	if cfg.Relays == nil {
		cfg.Relays = map[string]Relay{}
	}
	// Accept both {"blossom":{"servers":[...]}} and {"blossomServers":[...]} for a simpler config.
	if len(cfg.Blossom.Servers) == 0 && len(cfg.BlossomServers) > 0 {
		cfg.Blossom.Servers = cfg.BlossomServers
	}
	return &cfg, nil
}

func Save(cfg *Config) (string, error) {
	fp, err := ConfigPath()
	if err != nil {
		return "", err
	}
	if cfg.Relays == nil {
		cfg.Relays = map[string]Relay{}
	}
	if err := os.MkdirAll(filepath.Dir(fp), 0700); err != nil {
		return "", err
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(fp, append(b, '\n'), 0600); err != nil {
		return "", err
	}
	return fp, nil
}

func EnsureExample() (string, error) {
	fp, err := ConfigPath()
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(fp); err == nil {
		return fp, nil
	}
	if err := os.MkdirAll(filepath.Dir(fp), 0700); err != nil {
		return "", err
	}
	example := Config{
		Relays: map[string]Relay{
			"wss://relay-jp.nostr.wirednet.jp": {Read: true, Write: true, Search: false},
		},
		PrivateKey: "nsecXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
		Blossom:    Blossom{Servers: []string{"https://blossom.example.com"}},
		Nsite:      Nsite{Host: ""},
	}
	return Save(&example)
}

func (c *Config) SecretHex() (string, error) {
	prefix, v, err := nip19.Decode(c.PrivateKey)
	if err != nil {
		return "", err
	}
	if prefix != "nsec" {
		return "", errors.New("privatekey must be nsec")
	}
	sk, ok := v.(string)
	if !ok || sk == "" {
		return "", errors.New("invalid nsec payload")
	}
	return sk, nil
}

func (c *Config) WriteRelays() []string {
	out := []string{}
	for url, r := range c.Relays {
		if r.Write {
			out = append(out, url)
		}
	}
	sort.Strings(out)
	return out
}

func (c *Config) ReadRelays() []string {
	out := []string{}
	for url, r := range c.Relays {
		if r.Read {
			out = append(out, url)
		}
	}
	sort.Strings(out)
	return out
}

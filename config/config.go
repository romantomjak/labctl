package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/hcl/v2/hclsimple"
)

type Config struct {
	Kubernetes Kubernetes `hcl:"kubernetes,block"`
	Proxmox    Proxmox    `hcl:"proxmox,block"`
}

type Kubernetes struct {
	Dashboard KubernetesDashboard `hcl:"dashboard,block"`
	Node      Node                `hcl:"node,block"`
}

type Node struct {
	Name     string `hcl:"name,label"`
	Addr     string `hcl:"addr"`
	Username string `hcl:"username"`

	// Password is required if private key is not set.
	Password string `hcl:"password,optional"`

	// PrivateKeyFile is required if password is not set.
	PrivateKeyFile string `hcl:"private_key_file,optional"`

	// HostKey is required if password is not set.
	HostKey string `hcl:"host_key,optional"`
}

type KubernetesDashboard struct {
	Namespace string `hcl:"namespace"`
	User      string `hcl:"user"`
	URL       string `hcl:"url"`
}

type Proxmox struct {
	TimeoutRaw string `hcl:"timeout"`
	Timeout    time.Duration
	Nodes      []Node `hcl:"node,block"`
}

func FromFile(filename string) (*Config, error) {
	if strings.HasPrefix(filename, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home directory: %w", err)
		}
		filename = strings.Replace(filename, "~", home, 1)
	}

	cfg := &Config{}
	if err := hclsimple.DecodeFile(filename, nil, cfg); err != nil {
		return nil, fmt.Errorf("decode file: %w", err)
	}

	timeout, err := time.ParseDuration(cfg.Proxmox.TimeoutRaw)
	if err != nil {
		return nil, fmt.Errorf("parse timeout: %w", err)
	}

	cfg.Proxmox.Timeout = timeout

	return cfg, nil
}

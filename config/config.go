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
	SSH       KubernetesSSH       `hcl:"ssh,block"`
	Dashboard KubernetesDashboard `hcl:"dashboard,block"`
}

type KubernetesSSH struct {
	Username       string `hcl:"username"`
	Addr           string `hcl:"addr"`
	PrivateKeyFile string `hcl:"private_key_file"`
	HostKey        string `hcl:"host_key"`
}

type KubernetesDashboard struct {
	Namespace string `hcl:"namespace"`
	User      string `hcl:"user"`
	URL       string `hcl:"url"`
}

type Proxmox struct {
	TimeoutRaw string `hcl:"timeout"`
	Timeout    time.Duration
	Nodes      []ProxmoxNode `hcl:"node,block"`
}

type ProxmoxNode struct {
	Name     string `hcl:"name,label"`
	Username string `hcl:"username"`
	Password string `hcl:"password"`
	Addr     string `hcl:"addr"`
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

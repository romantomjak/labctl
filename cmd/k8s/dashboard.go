package k8s

import (
	"bytes"
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/romantomjak/labctl/config"
)

var dashboard = &cobra.Command{
	Use:          "dashboard",
	Short:        "Generate token and open dashboard",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.FromFile("~/.labctl.hcl")
		if err != nil {
			return fmt.Errorf("load configuration: %w", err)
		}

		token, err := kubectlCreateToken(cfg)
		if err != nil {
			return fmt.Errorf("kubectl: %w", err)
		}

		if err := pbcopy(token); err != nil {
			return fmt.Errorf("pbcopy: %w", err)
		}

		if err := exec.Command("open", cfg.Kubernetes.Dashboard.URL).Start(); err != nil {
			return fmt.Errorf("open: %w", err)
		}

		return nil
	},
}

func kubectlCreateToken(cfg *config.Config) ([]byte, error) {
	cmd := exec.Command("kubectl", "-n", cfg.Kubernetes.Dashboard.Namespace, "create", "token", cfg.Kubernetes.Dashboard.User)

	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("get stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start command: %w", err)
	}

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(out); err != nil {
		return nil, fmt.Errorf("read stdout: %w", err)
	}

	if err := out.Close(); err != nil {
		return nil, fmt.Errorf("close stdout: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("wait for command to exit: %w", err)
	}

	return buf.Bytes(), nil
}

func pbcopy(token []byte) error {
	cmd := exec.Command("pbcopy")

	in, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("get stdin pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start command: %w", err)
	}

	if _, err := in.Write(token); err != nil {
		return err
	}

	if err := in.Close(); err != nil {
		return fmt.Errorf("close stdin: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("wait for command to exit: %w", err)
	}

	return nil
}

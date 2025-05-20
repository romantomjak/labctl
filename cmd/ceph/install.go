package ceph

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/romantomjak/labctl/config"
	"github.com/spf13/cobra"
)

var install = &cobra.Command{
	Use:          "install",
	Short:        "Install cephadm locally",
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE:         installCommandFunc,
}

func installCommandFunc(cmd *cobra.Command, args []string) error {
	cfg, err := config.FromFile("~/.labctl.hcl")
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	if os.Geteuid() != 0 {
		return fmt.Errorf("this command must be run with sudo")
	}

	filename := cfg.Ceph.Cephadm

	// Check if we need to prompt to overwrite existing file.
	exists, err := fileExists(filename)
	if err != nil {
		return fmt.Errorf("stat file: %w", err)
	}

	if exists && !flagAssumeYes {
		answer, err := prompt(fmt.Sprintf("❓ Overwrite %s? (y/n) [n] ", filename))
		if err != nil {
			return err
		}

		switch strings.ToLower(answer) {
		case "y", "yes":
			break // nothing to do
		case "", "n", "no":
			fmt.Println("🙅‍♀️ Not overwriting file")
			return nil
		}
	}

	// Open file for writing.
	fout, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer fout.Close()

	// Chown the file with the user's UID/GID.
	uid, err := strconv.ParseInt(os.Getenv("SUDO_UID"), 10, 64)
	if err != nil {
		return fmt.Errorf("parse sudo uid: %w", err)
	}
	gid, err := strconv.ParseInt(os.Getenv("SUDO_GID"), 10, 64)
	if err != nil {
		return fmt.Errorf("parse sudo gid: %w", err)
	}

	if err := fout.Chown(int(uid), int(gid)); err != nil {
		return fmt.Errorf("chown file: %w", err)
	}

	// Download binary.
	url := "https://download.ceph.com/rpm-" + cfg.Ceph.Release + "/el9/noarch/cephadm"

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "labctl (+https://github.com/romantomjak/labctl)")

	client := &http.Client{
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: 3 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 3 * time.Second,
		},
		Timeout: 15 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(fout, resp.Body)
	if err != nil {
		return fmt.Errorf("download cephadm: %w", err)
	}

	fmt.Printf("✅ Installed at %s\n", filename)

	return nil
}

func fileExists(name string) (bool, error) {
	_, err := os.Stat(name)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func prompt(prompt string) (string, error) {
	fmt.Fprint(os.Stdout, prompt)

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("read input: %w", err)
	}

	return scanner.Text(), nil
}

package k8s

import (
	"bufio"
	"crypto/sha512"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/romantomjak/labctl/config"
	"github.com/romantomjak/labctl/ssh"
)

var backupExample = strings.Trim(`
  # Save backup to a given file
  labctl k8s backup /backup/etcd-snapshot.db

  # Compress backup with zstd
  labctl k8s backup --compress /backup/etcd-snapshot.db

  # Backup with desirable time format
  labctl k8s backup ~/Downloads/etcd-backup-$(date +%Y%m%d%H%M%S).db
`, "\n")

var backup = &cobra.Command{
	Use:          "backup [flags] <filename>",
	Short:        "Backup etcd cluster",
	Example:      backupExample,
	SilenceUsage: true,
	Args:         cobra.ExactArgs(1),
	RunE:         backupCommandFunc,
}

func backupCommandFunc(cmd *cobra.Command, args []string) error {
	cfg, err := config.FromFile("~/.labctl.hcl")
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	// Shell won't expand the path if argument is wrapped in quotes.
	filename, err := expandTilde(args[0])
	if err != nil {
		return err
	}

	// Check if we need to add zstd file extension.
	if flagCompressBackup && !strings.HasSuffix(filename, ".zst") {
		filename += ".zst"
	}

	// Check if the destination file already exists.
	_, err = os.Stat(filename)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	if err == nil {
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

	fmt.Println("🔒 Connecting to k8s")

	sshClient, err := ssh.New(cfg.Kubernetes.Node)
	if err != nil {
		return fmt.Errorf("ssh: %w", err)
	}
	defer sshClient.Close()

	fmt.Println("⚡️ Snapshotting etcd database")
	snapshot := fmt.Sprintf("/tmp/etcd-backup-%d.db", time.Now().UnixMilli())
	if err := sshClient.SnapshotETCD(snapshot); err != nil {
		return fmt.Errorf("snapshot etcd: %w", err)
	}

	if flagCompressBackup {
		fmt.Println("📦 Compressing snapshot with zstd")
		if err := sshClient.Compress(snapshot); err != nil {
			return fmt.Errorf("compress: %w", err)
		}

		// From here, snapshot filename should include the zstd file extension.
		snapshot += ".zst"
	}

	fmt.Println("⌛️ Downloading snapshot")
	if err := sshClient.Copy(snapshot, filename); err != nil {
		return fmt.Errorf("download snapshot: %w", err)
	}

	fmt.Println("🔍 Checking file integrity")
	remoteSHA512, err := sshClient.SHA512Sum(snapshot)
	if err != nil {
		return fmt.Errorf("remote sha512sum: %w", err)
	}

	localSHA512, err := sha512sum(filename)
	if err != nil {
		return fmt.Errorf("local sha512sum: %w", err)
	}

	if localSHA512 != remoteSHA512 {
		return fmt.Errorf("sha512 hashes are not matching")
	}

	// Cleanup remote
	if err := sshClient.Delete(snapshot); err != nil {
		return fmt.Errorf("delete remote copy: %w", err)
	}

	fmt.Printf("✅ Snapshot saved at %s\n", filename)

	return nil
}

func expandTilde(filename string) (string, error) {
	if !strings.HasPrefix(filename, "~") {
		return filename, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}

	return strings.Replace(filename, "~", home, 1), nil
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

func sha512sum(filename string) (string, error) {
	hash := sha512.New()

	f, err := os.Open(filename)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	_, err = io.Copy(hash, f)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

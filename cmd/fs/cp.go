package fs

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
	sdk "github.com/ucloud/ucloud-sandbox-sdk-go"
)

// remotePath describes a copy endpoint that may point at a sandbox.
type remotePath struct {
	sandboxID string // empty means the path is local
	path      string
}

func (p remotePath) isRemote() bool { return p.sandboxID != "" }

// parsePath splits a "<sandbox-id>:<path>" argument into its sandbox id and
// path. A path without a colon (or a Windows-style drive letter) is treated as
// local. The sandbox form requires a non-empty id and path.
func parsePath(arg string) (remotePath, error) {
	idx := strings.Index(arg, ":")
	// No colon, or a single-letter prefix that looks like a Windows drive:
	// treat as a local path.
	if idx <= 1 {
		return remotePath{path: arg}, nil
	}
	id := arg[:idx]
	p := arg[idx+1:]
	if p == "" {
		return remotePath{}, fmt.Errorf("invalid path %q: missing path after sandbox id", arg)
	}
	return remotePath{sandboxID: id, path: p}, nil
}

func newCpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cp <src-path> <dest-path>",
		Short: "Upload or download a file",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			src, err := parsePath(args[0])
			if err != nil {
				return err
			}
			dest, err := parsePath(args[1])
			if err != nil {
				return err
			}

			if src.isRemote() && dest.isRemote() {
				return fmt.Errorf("both paths refer to a sandbox: at most one path may include a sandbox id")
			}
			if !src.isRemote() && !dest.isRemote() {
				return fmt.Errorf("neither path refers to a sandbox: exactly one path must include a sandbox id")
			}

			cfg, err := config.Load()
			if err != nil {
				return err
			}
			client, err := config.NewClient(cfg)
			if err != nil {
				return err
			}

			ctx := context.Background()
			if src.isRemote() {
				return download(ctx, client, src, dest)
			}
			return upload(ctx, client, src, dest)
		},
	}
}

// download copies a file from a sandbox to the local filesystem.
func download(ctx context.Context, client *sdk.Client, src, dest remotePath) error {
	sbx, err := client.ConnectSandbox(ctx, src.sandboxID)
	if err != nil {
		return err
	}

	rc, err := sbx.Files.ReadStream(ctx, src.path)
	if err != nil {
		return err
	}
	defer rc.Close()

	// When the destination is an existing directory, keep the source file name.
	localPath := dest.path
	if info, err := os.Stat(localPath); err == nil && info.IsDir() {
		localPath = path.Join(localPath, path.Base(src.path))
	}

	f, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := io.Copy(f, rc); err != nil {
		return err
	}
	fmt.Printf("Downloaded %s:%s -> %s\n", src.sandboxID, src.path, localPath)
	return nil
}

// upload copies a local file to a sandbox.
func upload(ctx context.Context, client *sdk.Client, src, dest remotePath) error {
	sbx, err := client.ConnectSandbox(ctx, dest.sandboxID)
	if err != nil {
		return err
	}

	f, err := os.Open(src.path)
	if err != nil {
		return err
	}
	defer f.Close()

	// When the destination is a directory, keep the source file name.
	remoteDest := dest.path
	if strings.HasSuffix(remoteDest, "/") {
		remoteDest = path.Join(remoteDest, path.Base(src.path))
	}

	info, err := sbx.Files.WriteStream(ctx, remoteDest, f)
	if err != nil {
		return err
	}
	fmt.Printf("Uploaded %s -> %s:%s\n", src.path, dest.sandboxID, info.Path)
	return nil
}

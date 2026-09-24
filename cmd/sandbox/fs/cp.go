package fs

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
)

type cpOperation struct {
	session
}

func (o *cpOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "cp <src-path> <dest-path>",
		Short: "Upload or download a file",
		Args:  cobra.ExactArgs(2),
	}

	o.AddFlags(c.Flags())

	return c
}

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
	index := strings.Index(arg, ":")

	// No colon, or a single-letter prefix that looks like a Windows drive:
	// treat as a local path.
	if index <= 1 {
		return remotePath{path: arg}, nil
	}

	id, p := arg[:index], arg[index+1:]
	if p == "" {
		return remotePath{}, fmt.Errorf("invalid path %q: missing path after sandbox id", arg)
	}

	return remotePath{sandboxID: id, path: p}, nil
}

func (o *cpOperation) Run(ctx cmd.OperationContext) error {
	src, err := parsePath(ctx.Args[0])
	if err != nil {
		return err
	}

	dest, err := parsePath(ctx.Args[1])
	if err != nil {
		return err
	}

	// One side is the sandbox and the other the local machine: there is no
	// path from one sandbox straight into another.
	if src.isRemote() && dest.isRemote() {
		return fmt.Errorf("both paths refer to a sandbox: at most one path may include a sandbox id")
	}
	if !src.isRemote() && !dest.isRemote() {
		return fmt.Errorf("neither path refers to a sandbox: exactly one path must include a sandbox id")
	}

	if src.isRemote() {
		return o.download(ctx, src, dest)
	}

	return o.upload(ctx, src, dest)
}

// download copies a file from a sandbox to the local filesystem.
func (o *cpOperation) download(ctx cmd.OperationContext, src, dest remotePath) error {
	fs, err := o.open(ctx, src.sandboxID)
	if err != nil {
		return err
	}

	reader, err := fs.ReadStream(ctx, src.path)
	if err != nil {
		return err
	}
	defer reader.Close()

	// When the destination is an existing directory, keep the source file name.
	localPath := dest.path
	if info, err := os.Stat(localPath); err == nil && info.IsDir() {
		localPath = path.Join(localPath, path.Base(src.path))
	}

	file, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := io.Copy(file, reader); err != nil {
		return err
	}

	fmt.Printf("Downloaded %s:%s -> %s\n", src.sandboxID, src.path, localPath)

	return nil
}

// upload copies a local file to a sandbox.
func (o *cpOperation) upload(ctx cmd.OperationContext, src, dest remotePath) error {
	fs, err := o.open(ctx, dest.sandboxID)
	if err != nil {
		return err
	}

	file, err := os.Open(src.path)
	if err != nil {
		return err
	}
	defer file.Close()

	// When the destination names a directory, keep the source file name.
	remoteDest := dest.path
	if strings.HasSuffix(remoteDest, "/") {
		remoteDest = path.Join(remoteDest, path.Base(src.path))
	}

	info, err := fs.WriteStream(ctx, remoteDest, file)
	if err != nil {
		return err
	}

	fmt.Printf("Uploaded %s -> %s:%s\n", src.path, dest.sandboxID, info.Path)

	return nil
}

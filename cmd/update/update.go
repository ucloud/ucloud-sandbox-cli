package update

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/internal/prompt"
	"github.com/ucloud/ucloud-sandbox-cli/internal/selfupdate"
	"golang.org/x/term"
)

// Command returns the update command. version is the one stamped into this
// binary at build time.
func Command(version string) *cobra.Command {
	return cmd.BuildLocal(&updateOperation{version: version})
}

type updateOperation struct {
	version string

	dryRun bool
	yes    bool
}

func (o *updateOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "update",
		Short: "Update the CLI to the latest release",
		Args:  cobra.NoArgs,
	}

	c.Flags().BoolVarP(&o.dryRun, "dry-run", "", false,
		"Only check for a newer release and show what would be downloaded")
	c.Flags().BoolVarP(&o.yes, "yes", "y", false, "Don't ask for confirmation")

	return c
}

func (o *updateOperation) Run(ctx cmd.OperationContext) error {
	assetName, err := selfupdate.AssetName(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return err
	}

	executable, err := selfupdate.ExecutablePath()
	if err != nil {
		return err
	}

	fmt.Println("Checking for a newer release...")

	client := selfupdate.NewClient()
	release, err := client.LatestRelease(ctx)
	if err != nil {
		return err
	}

	displayedVersion := o.version
	if displayedVersion == "" {
		displayedVersion = "unknown"
	}
	fmt.Printf("Current version: %s\n", displayedVersion)
	fmt.Printf("Latest version:  %s\n", release.TagName)

	newer, err := selfupdate.IsNewer(o.version, release.TagName)
	if err != nil {
		if _, latestErr := selfupdate.Normalize(release.TagName); latestErr != nil {
			return latestErr
		}
		// A binary built outside of a release carries no comparable version,
		// so fall back to installing the latest release.
		fmt.Printf("Version %q is not a released version, updating to %s.\n", displayedVersion, release.TagName)
		newer = true
	}

	if !newer {
		fmt.Println("Already up to date.")
		return nil
	}

	asset, ok := release.Asset(assetName)
	if !ok {
		return fmt.Errorf("release %s does not provide %s", release.TagName, assetName)
	}

	// The checksum is what makes the download safe to run, so a release
	// without one is not installed.
	checksumAsset, ok := release.Asset(assetName + ".sha256")
	if !ok {
		return fmt.Errorf("release %s does not provide %s.sha256", release.TagName, assetName)
	}

	if o.dryRun {
		fmt.Println()
		fmt.Printf("Platform:     %s/%s\n", runtime.GOOS, runtime.GOARCH)
		fmt.Printf("Download URL: %s\n", asset.DownloadURL)
		fmt.Printf("Checksum URL: %s\n", checksumAsset.DownloadURL)
		fmt.Printf("Install path: %s\n", executable)
		fmt.Println()
		fmt.Println("Dry run, nothing was downloaded or installed.")
		return nil
	}

	if !o.yes {
		fmt.Printf("Install path:    %s\n", executable)

		confirmed, err := prompt.Confirm(fmt.Sprintf("Update %s to %s?", displayedVersion, release.TagName))
		if err != nil {
			return err
		}
		if !confirmed {
			return nil
		}
	}

	archive, err := download(ctx, client, asset)
	if err != nil {
		return err
	}

	fmt.Println("Verifying release SHA256...")

	checksum, err := client.DownloadBytes(ctx, checksumAsset.DownloadURL)
	if err != nil {
		return err
	}
	if err := selfupdate.VerifyChecksum(archive, checksum); err != nil {
		return err
	}

	binary, err := selfupdate.ExtractBinary(asset.Name, archive)
	if err != nil {
		return err
	}

	fmt.Printf("Installing %s to %s...\n", release.TagName, executable)
	err = selfupdate.Apply(binary, executable)
	if errors.Is(err, fs.ErrPermission) && selfupdate.CanSudo() {
		fmt.Printf("No permission to write to %s, retrying with sudo...\n", filepath.Dir(executable))
		err = selfupdate.ApplyWithSudo(ctx, binary, executable)
	}
	if err != nil {
		return err
	}

	fmt.Printf("Updated to %s.\n", release.TagName)

	return nil
}

// download fetches a release asset while rendering a progress bar.
func download(ctx context.Context, client *selfupdate.Client, asset selfupdate.Asset) ([]byte, error) {
	width, _, err := term.GetSize(int(os.Stderr.Fd()))
	// Leave room for byte counts, speed and ETA; use plain output when the
	// terminal cannot fit them or the output is being redirected.
	interactive := term.IsTerminal(int(os.Stderr.Fd())) && err == nil && width >= 80 && os.Getenv("TERM") != "dumb"
	return downloadWithProgress(ctx, client, asset, os.Stderr, interactive)
}

func downloadWithProgress(ctx context.Context, client *selfupdate.Client, asset selfupdate.Asset, out io.Writer, interactive bool) ([]byte, error) {
	// Keep the long asset name out of the line that is redrawn.
	fmt.Fprintf(out, "Downloading %s...\n", asset.Name)
	buf := &bytes.Buffer{}
	if !interactive {
		if err := client.Download(ctx, asset.DownloadURL, buf); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}

	bar := progressbar.NewOptions64(
		asset.Size,
		progressbar.OptionSetDescription("Downloading"),
		progressbar.OptionSetWriter(out),
		progressbar.OptionSetWidth(10),
		progressbar.OptionUseANSICodes(true),
		progressbar.OptionShowBytes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionThrottle(100*time.Millisecond),
		progressbar.OptionOnCompletion(func() { fmt.Fprintln(out) }),
	)

	if err := client.Download(ctx, asset.DownloadURL, io.MultiWriter(buf, bar)); err != nil {
		fmt.Fprintln(out)
		return nil, err
	}
	if err := bar.Finish(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

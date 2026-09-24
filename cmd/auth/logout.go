package auth

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
	"github.com/ucloud/ucloud-sandbox-cli/internal/config"
)

type logoutOperation struct{}

func (o *logoutOperation) Command() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove local credentials",
		Args:  cobra.NoArgs,
	}
}

func (o *logoutOperation) Run(_ cmd.OperationContext) error {
	path, err := config.Path()
	if err != nil {
		return err
	}

	// Being logged out already is what the caller asked for, not a failure.
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}

	fmt.Println("Logged out successfully.")

	return nil
}

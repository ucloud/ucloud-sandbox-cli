package sandbox

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/ucloud/ucloud-sandbox-cli/cmd"
)

type hostOperation struct {
	url bool
}

func (o *hostOperation) Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "host <sandbox-id> <port>",
		Short: "Print the address reaching a port inside a sandbox",
		Args:  cobra.ExactArgs(2),
	}

	c.Flags().BoolVarP(&o.url, "url", "", false, "Print a full URL rather than host:port")

	return c
}

func (o *hostOperation) Run(ctx cmd.OperationContext) error {
	port, err := strconv.Atoi(ctx.Args[1])
	if err != nil {
		return fmt.Errorf("invalid port %q: %w", ctx.Args[1], err)
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid port %d: must be between 1 and 65535", port)
	}

	// Read rather than connect: the address is derived from the sandbox's ID
	// and domain, so printing it has no reason to resume a paused sandbox.
	// The address of a paused one is still its address; it answers once the
	// sandbox is running again.
	detail, err := ctx.Client.Sandboxes().Get(ctx, ctx.Args[0])
	if err != nil {
		return err
	}

	domain := ""
	if detail.Domain != nil {
		domain = *detail.Domain
	}

	transport := ctx.Client.Transport()
	host := transport.SandboxHost(detail.SandboxID, domain, port)

	if o.url {
		host = transport.SandboxScheme() + "://" + host
	}

	fmt.Println(host)

	return nil
}

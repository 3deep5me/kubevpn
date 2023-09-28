package cmds

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/i18n"
	"k8s.io/kubectl/pkg/util/templates"

	"github.com/wencaiwulue/kubevpn/pkg/daemon"
	"github.com/wencaiwulue/kubevpn/pkg/daemon/rpc"
)

func CmdDisconnect(f cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disconnect",
		Short: i18n.T("Disconnect from kubernetes cluster network"),
		Long:  templates.LongDesc(i18n.T(`Disconnect from kubernetes cluster network`)),
		Example: templates.Examples(i18n.T(`
		# disconnect from cluster network and restore proxy resource
        kubevpn disconnect
`)),
		PreRunE: func(cmd *cobra.Command, args []string) (err error) {
			err = daemon.StartupDaemon(cmd.Context())
			return err
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := daemon.GetClient(false).Disconnect(
				cmd.Context(),
				&rpc.DisconnectRequest{},
			)
			var resp *rpc.DisconnectResponse
			for {
				resp, err = client.Recv()
				if err == io.EOF {
					break
				} else if err == nil {
					fmt.Fprint(os.Stdout, resp.Message)
				} else if code := status.Code(err); code == codes.DeadlineExceeded || code == codes.Canceled {
					break
				} else {
					return err
				}
			}
			fmt.Fprint(os.Stdout, "disconnect successfully")
			return nil
		},
	}
	return cmd
}

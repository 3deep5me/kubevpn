package cmds

import (
	"context"
	"fmt"
	"io"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/i18n"
	"k8s.io/kubectl/pkg/util/templates"

	"github.com/wencaiwulue/kubevpn/pkg/config"
	"github.com/wencaiwulue/kubevpn/pkg/daemon"
	"github.com/wencaiwulue/kubevpn/pkg/daemon/rpc"
	"github.com/wencaiwulue/kubevpn/pkg/handler"
	"github.com/wencaiwulue/kubevpn/pkg/util"
)

func CmdConnect(f cmdutil.Factory) *cobra.Command {
	var connect = &handler.ConnectOptions{}
	var sshConf = &util.SshConfig{}
	var transferImage, foreground bool
	cmd := &cobra.Command{
		Use:   "connect",
		Short: i18n.T("Connect to kubernetes cluster network"),
		Long:  templates.LongDesc(i18n.T(`Connect to kubernetes cluster network`)),
		Example: templates.Examples(i18n.T(`
		# Connect to k8s cluster network
		kubevpn connect

		# Connect to api-server behind of bastion host or ssh jump host
		kubevpn connect --ssh-addr 192.168.1.100:22 --ssh-username root --ssh-keyfile ~/.ssh/ssh.pem

		# it also support ProxyJump, like
		┌──────┐     ┌──────┐     ┌──────┐     ┌──────┐                 ┌────────────┐
		│  pc  ├────►│ ssh1 ├────►│ ssh2 ├────►│ ssh3 ├─────►... ─────► │ api-server │
		└──────┘     └──────┘     └──────┘     └──────┘                 └────────────┘
		kubevpn connect --ssh-alias <alias>

`)),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// startup daemon process and sudo process
			return daemon.StartupDaemon(cmd.Context())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			bytes, err := util.ConvertToKubeconfigBytes(f)
			if err != nil {
				return err
			}
			req := &rpc.ConnectRequest{
				KubeconfigBytes:  string(bytes),
				Namespace:        connect.Namespace,
				ExtraCIDR:        connect.ExtraCIDR,
				ExtraDomain:      connect.ExtraDomain,
				UseLocalDNS:      connect.UseLocalDNS,
				Engine:           string(connect.Engine),
				Addr:             sshConf.Addr,
				User:             sshConf.User,
				Password:         sshConf.Password,
				Keyfile:          sshConf.Keyfile,
				ConfigAlias:      sshConf.ConfigAlias,
				RemoteKubeconfig: sshConf.RemoteKubeconfig,
				TransferImage:    transferImage,
				Image:            config.Image,
				Level:            int32(log.DebugLevel),
			}
			cli := daemon.GetClient(false)
			stream, err := cli.Connect(cmd.Context(), req)
			if err != nil {
				return err
			}
			for {
				resp, err := stream.Recv()
				if err == io.EOF {
					break
				} else if err != nil {
					return err
				}
				log.Print(resp.GetMessage())
			}
			// hangup
			if foreground {
				// disconnect from cluster network
				<-cmd.Context().Done()

				now := time.Now()
				stream, err := cli.Disconnect(context.Background(), &rpc.DisconnectRequest{})
				fmt.Printf("call api disconnect use %s\n", time.Now().Sub(now).String())
				if err != nil {
					return err
				}
				var resp *rpc.DisconnectResponse
				for {
					resp, err = stream.Recv()
					if err == io.EOF {
						return nil
					} else if err == nil {
						fmt.Print(resp.Message)
					} else {
						return err
					}
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&config.Debug, "debug", false, "enable debug mode or not, true or false")
	cmd.Flags().StringVar(&config.Image, "image", config.Image, "use this image to startup container")
	cmd.Flags().StringArrayVar(&connect.ExtraCIDR, "extra-cidr", []string{}, "Extra cidr string, eg: --extra-cidr 192.168.0.159/24 --extra-cidr 192.168.1.160/32")
	cmd.Flags().StringArrayVar(&connect.ExtraDomain, "extra-domain", []string{}, "Extra domain string, the resolved ip will add to route table, eg: --extra-domain test.abc.com --extra-domain foo.test.com")
	cmd.Flags().BoolVar(&transferImage, "transfer-image", false, "transfer image to remote registry, it will transfer image "+config.OriginImage+" to flags `--image` special image, default: "+config.Image)
	cmd.Flags().BoolVar(&connect.UseLocalDNS, "use-localdns", false, "if use-lcoaldns is true, kubevpn will start coredns listen at 53 to forward your dns queries. only support on linux now")
	cmd.Flags().StringVar((*string)(&connect.Engine), "engine", string(config.EngineRaw), fmt.Sprintf(`transport engine ("%s"|"%s") %s: use gvisor and raw both (both performance and stable), %s: use raw mode (best stable)`, config.EngineMix, config.EngineRaw, config.EngineMix, config.EngineRaw))
	cmd.Flags().BoolVar(&foreground, "foreground", false, "foreground hang up")

	addSshFlags(cmd, sshConf)
	return cmd
}

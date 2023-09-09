package cmds

import (
	"errors"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/i18n"

	"github.com/wencaiwulue/kubevpn/pkg/daemon"
)

func CmdDaemon(_ cmdutil.Factory) *cobra.Command {
	var opt = &daemon.SvrOption{}
	cmd := &cobra.Command{
		Use:   "daemon",
		Short: i18n.T("Startup GRPC server"),
		Long:  i18n.T(`Startup GRPC server`),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			portPath := daemon.GetSockPath(opt.IsSudo)
			err := os.Remove(portPath)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
			pidPath := daemon.GetPidPath(opt.IsSudo)
			err = os.Remove(pidPath)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
			pid := os.Getpid()
			err = os.WriteFile(pidPath, []byte(strconv.Itoa(pid)), os.ModePerm)
			if err != nil {
				return err
			}
			err = os.Chmod(pidPath, os.ModePerm)
			return err
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			defer opt.Stop()
			return opt.Start(cmd.Context())
		},
		Hidden:                true,
		DisableFlagsInUseLine: true,
	}
	cmd.Flags().BoolVar(&opt.IsSudo, "sudo", false, "is sudo or not")
	return cmd
}

package plugins

import (
	"github.com/NTHU-LSALAB/riya-scheduler/pkg/plugins/riya"
	"github.com/spf13/cobra"
	"k8s.io/kubernetes/cmd/kube-scheduler/app"
)

func Register() *cobra.Command {
	return app.NewSchedulerCommand(
		app.WithPlugin(riya.Name,riya.New),
	)
}
package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"
	"k8s.io/klog"
	"k8s.io/kubernetes/cmd/kube-scheduler/app"

	"riya-scheduler/pkg/plugins" // TODO: chanage your scheduler name
)

func main() {
	// Initialize the log
	rand.Seed(time.Now().UTC().UnixNano())
	klog.InitFlags(nil)


	// register our plugin into kube-scheduler
	// refer: https://github.com/kubernetes/kubernetes/blob/v1.17.17/cmd/kube-scheduler/app/server.go
	cmd := app.NewSchedulerCommand(
		app.WithPlugin(plugins.Name, plugins.New),
	)

	if err := cmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

}
package main

import (
    "flag"
    "os"
    "os/signal"
    "syscall"

	"k8s.io/klog"
	 "github.com/NTHU-LSALAB/podMonitor-webhook/pkg/webhook"
)

var parameters webhook.WebHookServerParameters

func init() {
    //get command line parameters
    flag.IntVar(&parameters.Port, "port", 443, "The port of webhook server to listen.")
    flag.StringVar(&parameters.CertFile, "tlsCertPath", "/etc/webhook/certs/cert.pem", "The path of tls cert")
    flag.StringVar(&parameters.KeyFile, "tlsKeyPath", "/etc/webhook/certs/key.pem", "The path of tls key")
    flag.StringVar(&parameters.SidecarCfgFile, "sidecarCfgFile", "/etc/webhook/config/sidecarconfig.yaml", "File containing the mutation configuration.")
}


func main() {
    // parse parameters
    flag.Parse()

    // init webhook api
    ws, err := webhook.NewWebhookServer(parameters)
    if err != nil {
        panic(err)
    }

    // start webhook server in new routine
    go ws.Start()
    klog.Info("Server started")
    
    signalChan := make(chan os.Signal, 1)
    signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
    <-signalChan

    ws.Stop()
}




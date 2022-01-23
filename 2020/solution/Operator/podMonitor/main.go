package main

import (
	"flag"
	// "os"
	// "path/filepath"
	"time"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	clientset "github.com/NTHU-LSALAB/podMonitor/pkg/generated/clientset/versioned"
	informers "github.com/NTHU-LSALAB/podMonitor/pkg/generated/informers/externalversions"
	"github.com/NTHU-LSALAB/podMonitor/pkg/signals"
)

var (
	masterURL  string
	kubeconfig string
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	// create the actual Kubernetes client set
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	pmClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building podMonitor clientset: %s",err.Error())
	}

	// Informers are a combination of this event interface and an in-memory cache with indexed lookup. 
	// NewSharedInformerFactory caches all objects of a resource in all namespaces in the store
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	pmInformerFactory := informers.NewSharedInformerFactory(pmClient, time.Second*30)

	controller := NewController(kubeClient, pmClient,
		kubeInformerFactory.Core().V1().Pods(),
		pmInformerFactory.Lsalab().V1().PodMonitors())
	

	// notice that there is no need to run Start method in a separate goroutine.
	// (i.e. go kubeInformerFactory.Start(stopCh))
	// Start method is non-blocking and runs all registered informers in a dedicated goroutine.
	kubeInformerFactory.Start(stopCh)
	pmInformerFactory.Start(stopCh)

	if err = controller.Run(2, stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
}
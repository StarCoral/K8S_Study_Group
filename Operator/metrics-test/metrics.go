package main

import (
	
	"fmt"
	"os"
	"path/filepath"
	"time"
	"strconv"
	// "k8s.io/client-go/tools/clientcmd"
	// corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"
)

var (
	podmonitor_namespace string = "default"
	podmonitor_name      string
	podmonitor_logdir    string = "./log"
	podmonitor_speed	    int32 = 30
	podmonitor_logpath   string
)


func init() {
	podmonitor_namespace = os.Getenv("PODMONITOR_NAMESPACE")
	podmonitor_name = os.Getenv("PODMONITOR_NAME")
	podmonitor_logdir = os.Getenv("PODMONITOR_LOGDIR")
	speed_string := os.Getenv("PODMONITOR_SPEED")
	ps, err := strconv.ParseInt(speed_string, 10, 64)
	podmonitor_speed = int32(ps)
	if err != nil {
		klog.Info("podmonitor_speed change error, to be default 30 sec.")
	}
	if podmonitor_name == "" {
		klog.Fatalf("The Environmental variables are losts")
	}
	podmonitor_logpath = filepath.Join(podmonitor_logdir,"/",podmonitor_name)
	klog.Info("===Setting up Environmental variables ===")
	klog.Info("Pod is ",podmonitor_namespace," ",podmonitor_name)
	klog.Info("Log file will store in ",podmonitor_logpath)
	klog.Info("Sampling speed: ", podmonitor_speed, " sec.")
}

func main() {
	
	// create the file
	os.Mkdir(podmonitor_logdir, 0755)
	file, err := os.OpenFile(podmonitor_logpath, os.O_WRONLY|os.O_APPEND|os.O_CREATE,0600)
	if err != nil {
		fmt.Println("Can't open the file")
	}
	defer file.Close()

	msg := fmt.Sprintf("============ %s / %s ============ \n", podmonitor_namespace, podmonitor_name)
	klog.Info(msg)
	file.WriteString(msg)
	// klog.Info("============", podmonitor_namespace, " / ", podmonitor_name, "============")
	
	// get the config by the variables (KUBERNETES_SERVICE_HOST KUBERNETES_SERVICE_PORT)
	config, err := rest.InClusterConfig()
	if err != nil {
		klog.Fatalf("Can't get cluster config: %s", err.Error())
	}

	mc,err := metrics.NewForConfig(config)
	if err != nil {
			panic(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
			panic(err.Error())
	}
	
	for {
		pod, err := clientset.CoreV1().Pods(podmonitor_namespace).Get(podmonitor_name, metav1.GetOptions{})
		if err != nil {
			klog.Info(err)
			continue
		}

		msg := fmt.Sprintf("Pod status: %s\n", pod.Status.Phase)
		// klog.Info(" \t Pod status: ", pod.Status.Phase)
		klog.Info(msg)
		file.WriteString(msg)
		

		if pod.Status.Phase =="Succeeded" || pod.Status.Phase =="Failed"{
			klog.Info("Done.")
			file.WriteString("Done.\n")
			break
		}
		podMetrics, err := mc.MetricsV1beta1().PodMetricses(podmonitor_namespace).Get(podmonitor_name, metav1.GetOptions{})
		if err != nil {
			klog.Info(err)
		}
		for _, container := range podMetrics.Containers{
				cpuQuantity, ok := container.Usage.Cpu().AsInt64()
				memQuantity, ok := container.Usage.Memory().AsInt64()
				if !ok{
					return 
				}
				msg := fmt.Sprintf("Container Name: %s \t CPU(cores): %d \t MEMORY(bytes): %d", container.Name, cpuQuantity, memQuantity)
				klog.Infof(msg)
				// klog.Infof("Container Name: %s \t CPU usage: %d \t Memory usage: %d", container.Name, cpuQuantity, memQuantity)
				file.WriteString(msg)
		}
		time.Sleep(time.Duration(podmonitor_speed)*time.Second)
		
	}
	

}	

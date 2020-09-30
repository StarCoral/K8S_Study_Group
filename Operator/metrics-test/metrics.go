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
	podmontor_namespace string = "default"
	podmontor_name      string
	podmontor_logdir    string = "./log"
	podmontor_speed	    int32 = 30
	podmontor_logpath   string
)


func init() {
	podmontor_namespace = os.Getenv("PODMONTOR_NAMESPACE")
	podmontor_name = os.Getenv("PODMONTOR_NAME")
	podmontor_logdir = os.Getenv("PODMONTOR_LOGDIR")
	speed_string := os.Getenv("PODMONTOR_SPEED")
	ps, err := strconv.ParseInt(speed_string, 10, 64)
	podmontor_speed = int32(ps)
	if err != nil {
		klog.Info("podmontor_speed change error, to be default 30 sec.")
	}
	if podmontor_name == "" {
		klog.Fatalf("The Environmental variables are losts")
	}
	podmontor_logpath = filepath.Join(podmontor_logdir,"/",podmontor_name)
	klog.Info("===Setting up Environmental variables ===")
	klog.Info("Pod is ",podmontor_namespace," ",podmontor_name)
	klog.Info("Log file will store in ",podmontor_logpath)
	klog.Info("Sampling speed: ", podmontor_speed, " sec.")
}

func main() {
	
	// create the file
	os.Mkdir(podmontor_logdir, 0755)
	file, err := os.OpenFile(podmontor_logpath, os.O_WRONLY|os.O_APPEND|os.O_CREATE,0600)
	if err != nil {
		fmt.Println("Can't open the file")
	}
	defer file.Close()

	msg := fmt.Sprintf("============ %s / %s ============ \n", podmontor_namespace, podmontor_name)
	klog.Info(msg)
	file.WriteString(msg)
	// klog.Info("============", podmontor_namespace, " / ", podmontor_name, "============")
	
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
		pod, err := clientset.CoreV1().Pods(podmontor_namespace).Get(podmontor_name, metav1.GetOptions{})
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
		podMetrics, err := mc.MetricsV1beta1().PodMetricses(podmontor_namespace).Get(podmontor_name, metav1.GetOptions{})
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
		time.Sleep(time.Duration(podmontor_speed)*time.Second)
		
	}
	

}	

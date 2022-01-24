package plugins

import (
	"context"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	//"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog"

	framework "k8s.io/kubernetes/pkg/scheduler/framework/v1alpha1"
	//corelisters "k8s.io/client-go/listers/core/v1"
)

const (
	// plugin name
	Name = "riya" // TODO: modify your own plugin name
	
	// TODO: Label declaration & initialize
	groupName = "groupName"
	groupPriority = "groupPriority"
	minAvailable = "minAvailable"
)

var (
	// TODO: declare the plugin that you will implement
	// refer: https://github.com/kubernetes/kubernetes/blob/v1.17.17/pkg/scheduler/framework/v1alpha1/interface.go
	_ framework.QueueSortPlugin = &RiyaScheduler{}
	_ framework.PreFilterPlugin = &RiyaScheduler{}
)

type Args struct {
	kubeConfig string `json:"kubeconfig,omitempty"`
	master string `json:"master,omitempty"`
}

// TODO: define your plugin structure 
type RiyaScheduler struct {

}


func New(configuration *runtime.Unknown, f framework.FrameworkHandle) (framework.Plugin, error) {
	args := &Args{}
	if err := framework.DecodeInto(configuration, args); err != nil {
		return nil, err
	}
	klog.Infof("Get plugin config args: %+v",args)


	// TODO: construct your own plugin 
	

	
	return &RiyaScheduler{
		
	}, nil
}

//TODO: modify your own plugin method Name
func (r *RiyaScheduler) Name() string {
	return Name
}

//TODO: implement your own plugin method
func (r *RiyaScheduler) Less(podInfo1, podInf2 *framework.PodInfo) bool {
	return true
}

//TODO: implement your own plugin method
func (r *RiyaScheduler) PreFilter(ctx context.Context, state *framework.CycleState, pod *v1.Pod) *framework.Status {
	return framework.NewStatus(framework.Success, "")
}

//TODO: implement your own plugin method
func (r *RiyaScheduler) PreFilterExtensions() framework.PreFilterExtensions {
	return nil
}
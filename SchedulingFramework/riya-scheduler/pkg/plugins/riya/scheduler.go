package riya

import (
	"context"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog"
	framework "k8s.io/kubernetes/pkg/scheduler/framework/v1alpha1"
	corelisters "k8s.io/client-go/listers/core/v1"
	//"k8s.io/kubernetes/pkg/scheduler/nodeinfo"
	
	// custom
	"github.com/NTHU-LSALAB/riya-scheduler/pkg/plugins/riya/queueSort"
	"github.com/NTHU-LSALAB/riya-scheduler/pkg/plugins/riya/prefilter"
)

const (
	// Name is plugin name
	Name = "riya"

	PodGroupName = "podGroup"
	PodGroupMinAvailable = "minAvailable"
)

var (
	_ framework.QueueSortPlugin = &Riya{}
	_ framework.PreFilterPlugin = &Riya{}
)

type Args struct {
	KubeConfig string `json:"kubeconfig,omitempty"`
	Master string `json:"master,omitempty"`
}

type Riya struct {
	args *Args
	handle framework.FrameworkHandle

	//
	podLister corelisters.PodLister
}

func New(configuration *runtime.Unknown, f framework.FrameworkHandle) (framework.Plugin, error) {
	args := &Args{}
	if err := framework.DecodeInto(configuration, args); err != nil {
		return nil, err
	}
	klog.V(3).Infof("Get plugin config args: %+v",args)

	podLister := f.SharedInformerFactory().Core().V1().Pods().Lister()
	return &Riya{
		args:	args,
		handle:	f,
		podLister: podLister,
	}, nil
}

func (r *Riya) Name() string {
	return Name
}

// queueSort
func (r *Riya) Less(podInfo1, podInfo2 *framework.PodInfo) bool {
	klog.V(3).Info("---QueueSort---")
	return queueSort.Less(podInfo1, podInfo2)
}


// preFilter
func (r *Riya) PreFilter(ctx context.Context, state *framework.CycleState, pod *v1.Pod) *framework.Status {
	klog.V(3).Info("---PreFilter---")
	podGroupName, minAvailable , err := prefilter.GetPodGroupLabels(pod)
	if err != nil {
		return framework.NewStatus(framework.Error, err.Error())
	}

	if podGroupName == "" || minAvailable <= 1 {
		return framework.NewStatus(framework.Success, "")
	}

	total := r.calculateTotalPods(podGroupName, pod.Namespace)
	if total < minAvailable {
		klog.V(3).Infof("The count of podGroup %v/%v/%v is not up to minAvailable(%d) in PreFilter: %d",
			pod.Namespace, podGroupName, pod.Name, minAvailable, total)
		return framework.NewStatus(framework.Unschedulable, "less than minAvailable")
	}
	
	return framework.NewStatus(framework.Success, "")
}

func (r *Riya) PreFilterExtensions() framework.PreFilterExtensions {
	return nil
}

// func (r *Riya) AddPod(ctx context.Context, state *framework.CycleState, podToSchedule *v1.Pod, podToAdd *v1.Pod, nodeInfo *schedulernodeinfo.NodeInfo) *Status {
// 	return nil
// }

// func (r *Riya) RemovePod(ctx context.Context, state *framework.CycleState, podToSchedule *v1.Pod, podToRemove *v1.Pod, nodeInfo *schedulernodeinfo.NodeInfo) *Status {
// 	return nil
// }

func (r *Riya) calculateTotalPods(podGroupName, namespace string) int {
	// TODO get the total pods from the scheduler cache and queue instead of the hack manner
	selector := labels.Set{PodGroupName: podGroupName}.AsSelector()
	pods, err := r.podLister.Pods(namespace).List(selector)
	if err != nil {
		klog.Error(err)
		return 0
	}
	return len(pods)
}

// func (r *Riya) calculateRunningPods(podGroupName, namespace string) int {
// 	pods, err := r.handle.SnapshotSharedLister().Pods().FilteredList(func(pod *v1.Pod) bool {
// 		if pod.Labels[PodGroupName] == podGroupName && pod.Namespace == namespace && pod.Status.Phase == v1.PodRunning {
// 			return true
// 		}
// 		return false
// 	}, labels.NewSelector())
	
// 	if err != nil {
// 		klog.Error(err)
// 		return 0
// 	}
// 	return len(pods)
// }
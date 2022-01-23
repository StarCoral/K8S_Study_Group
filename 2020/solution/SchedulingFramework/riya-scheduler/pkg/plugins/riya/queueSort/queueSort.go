package queueSort

import (
	"strconv"
	framework "k8s.io/kubernetes/pkg/scheduler/framework/v1alpha1"
	//"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/kubernetes/pkg/apis/core/v1/helper/qos"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)



func Less(podInfo1, podInfo2 *framework.PodInfo) bool {
	pw1 := getPodPriority(podInfo1) 
	pw2 := getPodPriority(podInfo2)
	klog.V(3).Infof("The group Priority %v/%v is  %d",	podInfo1.Pod.Namespace, podInfo1.Pod.Name, pw1)
	klog.V(3).Infof("The group Priority %v/%v is  %d",	podInfo2.Pod.Namespace, podInfo2.Pod.Name, pw2)
	
	klog.V(3).Infof("The group Priority %v/%v is  greater than the group Priority %v/%v ? %v",	podInfo1.Pod.Namespace, podInfo1.Pod.Name, podInfo2.Pod.Namespace, podInfo2.Pod.Name, (pw1 > pw2) )
	klog.V(3).Infof("QOS %v vs %v : %v", podInfo1.Pod.Name, podInfo2.Pod.Name,compareQOS(podInfo1.Pod, podInfo2.Pod))
	return (pw1 > pw2) || (pw1 == pw2 && compareQOS(podInfo1.Pod, podInfo2.Pod))
}

func getPodPriority(podInfo *framework.PodInfo) int64 {
	var pod int64 = 0.0
	if val, ok := podInfo.Pod.Labels["groupPriority"]; ok {
		pod,_ = strconv.ParseInt(val, 10, 64)
	}
	return pod
}

func compareQOS(p1, p2 *v1.Pod) bool {
	pq1, pq2 := qos.GetPodQOS(p1), qos.GetPodQOS(p2)
	klog.V(3).Infof("The QOS %v/%v is %v",	p1.Namespace, p1.Name, pq1)
	klog.V(3).Infof("The QOS %v/%v is %v",	p2.Namespace, p2.Name, pq2)
	if pq1 == v1.PodQOSGuaranteed {
		return true
	} else if pq1 == v1.PodQOSBurstable {
		return pq2 != v1.PodQOSGuaranteed
	} else {
		return pq2 == v1.PodQOSBestEffort
	}
}
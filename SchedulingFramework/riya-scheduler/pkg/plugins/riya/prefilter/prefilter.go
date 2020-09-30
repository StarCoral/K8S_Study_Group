package prefilter

import (
	"strconv"
	
	"k8s.io/klog"
	v1 "k8s.io/api/core/v1"

)

const (
	PodGroupName = "podGroup"
	PodGroupMinAvailable = "minAvailable"
)

// GetPodGroupLabels will check the pod if belongs to some podGroup. If so, it will return the
// podGroupName、minAvailable of podGroup. If not, it will return "" as podGroupName.
func GetPodGroupLabels(pod *v1.Pod) (string, int, error) {
	podGroupName, exist := pod.Labels[PodGroupName]
	if !exist || podGroupName == "" {
		return "", 0, nil
	}
	minAvailable, exist := pod.Labels[PodGroupMinAvailable]
	if !exist || minAvailable == "" {
		return "", 0, nil
	}
	minNum, err := strconv.Atoi(minAvailable)
	if err != nil {
		klog.Errorf("GetPodGroupLabels err in riya-schduling %v/%v : %v", pod.Namespace, pod.Name, err.Error())
		return "", 0, nil
	}
	return podGroupName, minNum, nil
}


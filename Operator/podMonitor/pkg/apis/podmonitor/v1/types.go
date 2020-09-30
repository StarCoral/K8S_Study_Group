package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type PodMonitor struct {
	metav1.TypeMeta		`json:",inline"`
	metav1.ObjectMeta	`json:"metadata,omitempty"`

	Spec	PodMonitorSpec		`json:"spec"`
	Status	PodMonitorStatus	`json:"status"`
}

type PodMonitorPhase string

const (
	PodMonitorNone			PodMonitorPhase = ""
	PodMonitorCreating		PodMonitorPhase = "Creating"
	PodMonitorPending		PodMonitorPhase = "Pending"
	PodMonitorRunning		PodMonitorPhase = "Running"
	PodMonitorTerminating	PodMonitorPhase = "Terminating"
	PodMonitorCompleted		PodMonitorPhase = "Completed"
	PodMonitorFailed		PodMonitorPhase = "Failed"
	// PodMonitorUnknown		PodMonitorPhase = "Unknown"
)

type PodMonitorSpec struct {
	//	the default second is 30 sec
	Speed	int32  `json:"speed,omitempty"`
	// the file that store the information about resource usage
	LogDir	string `json:"logdir,omitempty"`
}

type PodMonitorStatus struct {
	Phase	PodMonitorPhase	`json:"phase"`
	Reason  string          `json:"reason,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type PodMonitorList struct {
	metav1.TypeMeta		`json:",inline"`
	metav1.ListMeta 	`json:"metadata"`

	Items []PodMonitor	`json:"items"`
}
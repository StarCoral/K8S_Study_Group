/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"reflect"
	"strconv"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	lsalabnthuv1 "github.com/NTHU-LSALAB/podMonitor-operator/apis/lsalab.nthu/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	PodMonitorLogPath = "/var/local"
)

// PodMonitorReconciler reconciles a PodMonitor object
type PodMonitorReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=lsalab.nthu.lsalab.nthu,resources=podmonitors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=lsalab.nthu.lsalab.nthu,resources=podmonitors/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=pods,verbs=list;watch;create;update;patch;delete

func (r *PodMonitorReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	//log := r.Log.WithValues("podmonitor", req.NamespacedName)
	_ = r.Log.WithValues("podmonitor", req.NamespacedName)
	// your logic here

	// Fetch the PodMonitor instance
	pm := &lsalabnthuv1.PodMonitor{}
	err := r.Get(ctx, req.NamespacedName, pm)
	if err != err {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			//log.Info("PodMonitor resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}

		// Error reading the object - requeue the request.
		//log.Error(err, "Failed to get PodMonitor")
		return ctrl.Result{}, err
	}
	pm_pod := pm.DeepCopy()
	//key := pm.Namespace + "-" + pm.Name
	switch pm.Status.Phase {
	case lsalabnthuv1.PodMonitorNone:
		//log.Info("-------------PodMonitorNone-------------")
		if err := r.changePhase(pm, lsalabnthuv1.PodMonitorPending, ctx); err != nil {
			return ctrl.Result{}, err
		}
		klog.Infof("Current phase: %v", pm.Status.Phase)
	case lsalabnthuv1.PodMonitorPending, lsalabnthuv1.PodMonitorFailed:
		//log.Info("PodMonitor ", key, ": phase=PENDING")
		// Check if the pod that needs to be monitored already exists
		found := &corev1.Pod{}
		err := r.Get(ctx, types.NamespacedName{Name: pm.Name, Namespace: pm.Namespace}, found)
		if errors.IsNotFound(err) {
			//log.Info("Can't find the pod to monitor.")
			r.changePhase(pm, lsalabnthuv1.PodMonitorFailed, ctx)
			return ctrl.Result{Requeue: true}, nil
		}

		if err := r.changePhase(pm, lsalabnthuv1.PodMonitorRunning, ctx); err != nil {
			return ctrl.Result{}, err
		}
	case lsalabnthuv1.PodMonitorRunning:
		//log.Info("PodMonitor ", key, ": phase=RUNNING")
		pod := createPodForCRD(pm)

		// Set the pm instance as the owner and controller
		// owner := metav1.NewControllerRef(pm_pod, lsalabnthuv1.GroupVersion.WithKind("PodMonitor"))
		// pod.ObjectMeta.OwnerReferences = append(pod.ObjectMeta.OwnerReferences, *owner)

		if err := ctrl.SetControllerReference(pm_pod, pod, r.Scheme); err != nil {
			//log.Error(err, "unable to set pod's owner reference")
			return ctrl.Result{}, err
		}

		found := corev1.Pod{}

		err := r.Get(ctx, types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, &found)
		//klog.Infof("PM NAME: ")
		if err != nil && errors.IsNotFound(err) {
			err = r.Create(ctx, pod) //Client.Status
			if err != nil {
				//log.Error(err, "unable to create pod")
				return ctrl.Result{}, err
			}
			//log.Info("PodMonitor ", key, ": pod launched: name=", pod.Name)
		} else if err != nil {
			//log.Info("Get Eroooooor.....")
			return ctrl.Result{}, err
		} else if found.Status.Phase == corev1.PodFailed || found.Status.Phase == corev1.PodSucceeded {
			//log.Info("PodMonitor ", key, ": container terminated: reason=", found.Status.Reason, " message=", found.Status.Message)
			pm_pod.Status.Phase = lsalabnthuv1.PodMonitorCompleted
		} else {
			klog.Infof("What happen: %v", ctrl.Result{})
			return ctrl.Result{}, nil
		}
	case lsalabnthuv1.PodMonitorCompleted:
		//log.Info("PodMonitor ", key, ": phase=COMPLETED")
		return ctrl.Result{}, nil
	default:
		//log.Info("PodMonitor ", key, ": NOP")
		return ctrl.Result{Requeue: true}, nil
	}

	if !reflect.DeepEqual(pm, pm_pod) {
		err := r.Client.Status().Update(ctx, pm_pod)
		if err != nil {
			//log.Error(err, "Failed to update pm_pod status")
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{Requeue: true}, nil
}

func (r *PodMonitorReconciler) changePhase(pm *lsalabnthuv1.PodMonitor, phase lsalabnthuv1.PodMonitorPhase, ctx context.Context) error {
	r.Log.Info("-------------changePhase-------------")
	pmCopy := pm.DeepCopy()
	return r.updateStatus(pmCopy, phase, ctx, nil)
	//return r.updateStatus(pm, phase, ctx, nil)
}

func (r *PodMonitorReconciler) updateStatus(pm *lsalabnthuv1.PodMonitor, phase lsalabnthuv1.PodMonitorPhase, ctx context.Context, reason error) error {
	r.Log.Info("-------------updateStatus-------------")
	pm.Status.Reason = ""
	if reason != nil {
		pm.Status.Reason = reason.Error()
	}

	pm.Status.Phase = phase
	err := r.Client.Status().Update(ctx, pm)
	if err != nil {
		r.Log.Error(err, "Upadate Failed")
	}
	klog.Infof("Current phase: %v", pm.Status.Phase)
	return err
}

func createPodForCRD(pm *lsalabnthuv1.PodMonitor) *corev1.Pod {
	klog.Info("Creating the crd pod to monitor...")
	labels := map[string]string{
		"podmonitors.lsalab.nthu": pm.Name,
		"podmonitors":             "true",
	}

	PodMonitorLogDir := pm.Spec.LogDir

	// This is the template
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "podmonitor-log",
			Namespace: pm.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "podmonitor",
					Image: "riyazhu/podmonitor:latest",
				},
			},
			RestartPolicy:      corev1.RestartPolicyOnFailure,
			ServiceAccountName: "podmonitor-metricstest",
		},
	}

	podTem := pod.Spec.DeepCopy()

	// add environment variable & mount volumn
	for i := range podTem.Containers {
		c := &podTem.Containers[i]
		c.Env = append(c.Env,
			corev1.EnvVar{
				Name:  "PODMONITOR_NAMESPACE",
				Value: pm.Namespace,
			},
			corev1.EnvVar{
				Name:  "PODMONITOR_NAME",
				Value: pm.Name,
			},
			corev1.EnvVar{
				Name:  "PODMONITOR_SPEED",
				Value: strconv.Itoa(int(pm.Spec.Speed)),
			},
			corev1.EnvVar{
				Name:  "PODMONITOR_LOGDIR",
				Value: pm.Spec.LogDir,
			},
		)
		c.VolumeMounts = append(c.VolumeMounts,
			corev1.VolumeMount{
				Name:      "podmonitor-log",
				MountPath: PodMonitorLogDir,
			},
		)
	}

	podTem.Volumes = append(podTem.Volumes,
		corev1.Volume{
			Name: "podmonitor-log",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: PodMonitorLogPath,
				},
			},
		},
	)

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pm-" + pm.Name,
			Namespace: pm.Namespace,
			Labels:    labels,
		},
		Spec: *podTem,
	}
}

func (r *PodMonitorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&lsalabnthuv1.PodMonitor{}).
		// Owns(&corev1.Pod{}).
		Complete(r)
}

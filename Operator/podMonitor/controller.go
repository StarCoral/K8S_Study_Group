package main

import (
	"fmt"
	"reflect"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// "k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	corev1informer "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	corev1lister "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"

	lsalabv1 "github.com/NTHU-LSALAB/podMonitor/pkg/apis/podmonitor/v1"
	clientset "github.com/NTHU-LSALAB/podMonitor/pkg/generated/clientset/versioned"
	pmscheme "github.com/NTHU-LSALAB/podMonitor/pkg/generated/clientset/versioned/scheme"
	informers "github.com/NTHU-LSALAB/podMonitor/pkg/generated/informers/externalversions/podmonitor/v1"
	listers "github.com/NTHU-LSALAB/podMonitor/pkg/generated/listers/podmonitor/v1"
)

const controllerAgentName = "PodMonitorController"

const (
	PodMonitorLogPath = "/var/local"
)

// Controller is the controller implementation for PodMonitor resources
type Controller struct {
	kubeClientset kubernetes.Interface
	pmClientset   clientset.Interface

	podLister   corev1lister.PodLister
	podsSynced  cache.InformerSynced
	pmLister	listers.PodMonitorLister
	pmsSynced	cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

// NewController returns a new pm controller
func NewController(
	kubeClientset kubernetes.Interface,
	pmClientset	  clientset.Interface,
	podInformer corev1informer.PodInformer,
	pmInformer	informers.PodMonitorInformer) *Controller {

	// Create event broadcaster
	// Add PodMonitorController types to the default 
	// kubernetes Scheme so Events can be logged for 
	// PodMonitorController types.
	utilruntime.Must(pmscheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(klog.Infof)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeClientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeClientset:		kubeClientset,
		pmClientset:		pmClientset,
		podLister:			podInformer.Lister(),
		podsSynced:			podInformer.Informer().HasSynced,
		pmLister:			pmInformer.Lister(),
		pmsSynced:			pmInformer.Informer().HasSynced,
		workqueue:     		workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "PodMonitors"),
		recorder:      		recorder,
	}

	klog.Info("Setting up event handlers")
	// Set up an event handler for when PodMonitor resources change
	pmInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:	controller.enqueuePodMonitor,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueuePodMonitor(new)
		},
	})

	podInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueuePod,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueuePod(new)
		},
	})
	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <- chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting PodMonitor controller")

	// Wait for the caches to be synced before starting workers
	if ok := cache.WaitForCacheSync(stopCh, c.pmsSynced); !ok {
		return fmt.Errorf("faild to wait for caches to sync")
	}

	if ok := cache.WaitForCacheSync(stopCh, c.podsSynced); !ok {
		return fmt.Errorf("faild to wait for caches to sync")
	}

	klog.Info("Starting workers")
	// Launch two workers to process PodMonitor resources
	for i := 0 ; i < threadiness ; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown{
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string 
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string) ; !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}

		// Run the syncHandler, passing it the namespace/name string of the 
		// PodMonitor resource to be synced.
		if err := c.syncHandler(key); err != nil{
			// Put the item back on the workqueue to handle any transient errors.
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s,requeuing",key,err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		klog.Infof("Successfully Synced '%s'",key)
		return nil

	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the PodMonitor resource
// with the current status of the resource. It returns how long to wait
// until the schedule is due.
func (c *Controller) syncHandler(key string) (error) {
	klog.Infof("------------- Reconciling Resource PodMonitor %s -------------", key)

	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the PodMonitor resource with this namespace/name
	pm, err := c.pmLister.PodMonitors(namespace).Get(name)
	if err != nil {
		// The PodMonitor resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("pm '%s' in work queue no longer exists", key))
			return  nil
		}
	}
	
	pm_pod := pm.DeepCopy()

	switch pm.Status.Phase {
	case lsalabv1.PodMonitorNone:
		if err := c.changePhase(pm, lsalabv1.PodMonitorPending); err != nil {
			return err
		}
	case lsalabv1.PodMonitorPending, lsalabv1.PodMonitorFailed:
		klog.Infof("PodMonitor %s: phase=PENDING", key)
		//pod := createPodForCRD(pm)
		_, err := c.podLister.Pods(namespace).Get(name)
		if errors.IsNotFound(err) {
			klog.Info("Can't find the pod to monitor.")
			return err
		}
		 
		if err := c.changePhase(pm, lsalabv1.PodMonitorRunning); err != nil {
			return err
		}
	case lsalabv1.PodMonitorRunning:
		klog.Infof("PodMonitor %s: phase=RUNNING", key)
		pod := createPodForCRD(pm)
		// Set the pm instance as the owner and controller
		owner := metav1.NewControllerRef(pm_pod, lsalabv1.SchemeGroupVersion.WithKind("PodMonitor"))
		pod.ObjectMeta.OwnerReferences = append(pod.ObjectMeta.OwnerReferences, *owner)

		found, err :=  c.kubeClientset.CoreV1().Pods(pod.Namespace).Get(pod.Name, metav1.GetOptions{})
		if err != nil && errors.IsNotFound(err) {
			found, err = c.kubeClientset.CoreV1().Pods(pod.Namespace).Create(pod)
			if err != nil {
				return err
			}
			klog.Infof("PodMonitor %s: pod launched: name=%s", key, pod.Name)
		}else if err != nil {
			return err
		}else if found.Status.Phase == corev1.PodFailed || found.Status.Phase == corev1.PodSucceeded {
			klog.Infof("PodMonitor %s: container terminated: reason=%q message=%q", key, found.Status.Reason, found.Status.Message)
			pm_pod.Status.Phase = lsalabv1.PodMonitorCompleted
		} else {
			return nil
		}
	case lsalabv1.PodMonitorCompleted:
		klog.Infof("PodMonitor %s: phase=COMPLETED", key)
		return nil
	default:
		klog.Infof("PodMonitor %s: NOP", key)
		return nil
	}
	if !reflect.DeepEqual(pm, pm_pod) {
		_, err := c.pmClientset.LsalabV1().PodMonitors(pm_pod.Namespace).UpdateStatus(pm_pod)
		if err != nil {
			return err
		}
	}
	return nil
}
// enqueuePodMonitor takes a PodMonitor resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than PodMonitor.
func (c *Controller) enqueuePodMonitor(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

func (c *Controller) enqueuePod(obj interface{}) {
	var pod *corev1.Pod
	var ok bool
	if pod, ok = obj.(*corev1.Pod); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding pod, invalid type"))
			return
		}
		pod, ok = tombstone.Obj.(*corev1.Pod)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding pod tombstone, invalid type"))
			return
		}
		klog.V(4).Infof("Recovered deleted pod '%s' from tombstone", pod.GetName())
	}
	if ownerRef := metav1.GetControllerOf(pod); ownerRef != nil {
		if ownerRef.Kind != "PodMonitor" {
			return
		}

		pm, err := c.pmLister.PodMonitors(pod.GetNamespace()).Get(ownerRef.Name)
		if err != nil {
			klog.V(4).Infof("ignoring orphaned pod '%s' of PodMonitor '%s'", pod.GetSelfLink(), ownerRef.Name)
			return
		}

		klog.Infof("enqueuing PodMonitor %s/%s because pod changed", pm.Namespace, pm.Name)
		c.enqueuePodMonitor(pm)
	}
}

func (c *Controller) changePhase(pm *lsalabv1.PodMonitor, phase lsalabv1.PodMonitorPhase) error {
	// Clone because the pm object is owned by the lister
	pmCopy := pm.DeepCopy()
	return c.updateStatus(pmCopy, phase, nil)
}

func (c *Controller) updateStatus(pm *lsalabv1.PodMonitor, phase lsalabv1.PodMonitorPhase, reason error) error {
	pm.Status.Reason = ""
	if reason != nil {
		pm.Status.Reason = reason.Error()
	}

	pm.Status.Phase = phase
	_, err := c.pmClientset.LsalabV1().PodMonitors(pm.Namespace).Update(pm)
	return err
}

func createPodForCRD(pm *lsalabv1.PodMonitor) *corev1.Pod {
	labels := map[string]string {
		"podmonitors.lsalab.nthu": pm.Name,
		"podmonitors": "true",
	}
	PodMonitorLogDir := pm.Spec.LogDir
	//var pod *corev1.Pod
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:	"podmonitor-log",
			Namespace: pm.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:	"podmonitor",
					Image:  "riyazhu/podmonitor:latest",
					// Name:    "busybox",
					// Image:   "busybox",
				},
			},
			RestartPolicy: corev1.RestartPolicyOnFailure,
			ServiceAccountName: "podmonitor-metricstest",
		},
	}
	//klog.Info("There is pod template ", pod)
	podTem := pod.Spec.DeepCopy()
	
	for i := range podTem.Containers{
		c := &podTem.Containers[i]
		c.Env = append( c.Env,
				corev1.EnvVar{
					Name: "PODMONTOR_NAMESPACE",
					Value: pm.Namespace,
				},
				corev1.EnvVar{
					Name: "PODMONTOR_NAME",
					Value: pm.Name,
				},
				corev1.EnvVar{
					Name: "PODMONTOR_SPEED",
					Value: strconv.Itoa(int(pm.Spec.Speed)),
				},
				corev1.EnvVar{
					Name: "PODMONTOR_LOGDIR",
					Value: pm.Spec.LogDir,
				},
		)
		c.VolumeMounts = append(c.VolumeMounts,
			corev1.VolumeMount{
				Name:	"podmonitor-log",
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
			Name:	"pm-"+pm.Name,
			Namespace: pm.Namespace,
			Labels: labels,
		},
		Spec: *podTem,
		// Spec: corev1.PodSpec{
		// 	Containers:	[]corev1.Container{
		// 		{
		// 			Name:	"podmonitor",
		// 			Image:  "riyazhu/podmonitor:latest",
		// 			Env:    env,
		// 			Volumes: Volumes,
		// 			VolumeMounts: VolumeMounts,
		// 			// Command: , //
		// 		},
		// 	},
		// 	RestartPolicy: corev1.RestartPolicyOnFailure,
		// },
	}
}
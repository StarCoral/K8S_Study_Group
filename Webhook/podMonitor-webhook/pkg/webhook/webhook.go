package webhook

import (
    "sync"
    "fmt"
    "context"
    "reflect"
    "strings"
    "net/http"
    "encoding/json"
    "crypto/sha256"
    "crypto/tls"
    "io/ioutil"

    "k8s.io/klog"
    "k8s.io/apimachinery/pkg/runtime"
    "k8s.io/apimachinery/pkg/runtime/serializer"
    corev1 "k8s.io/api/core/v1"
    "k8s.io/api/admission/v1beta1"
    admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/kubernetes/pkg/apis/core/v1"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/rest"

   "github.com/ghodss/yaml"
)

const (
    admissionWebhookAnnotationValidateKey = "podmonitor-webhook.nthu.lsalab/validate"
    admissionWebhookAnnotationMutateKey   = "podmonitor-webhook.nthu.lsalab/mutate"
    admissionWebhookAnnotationStatusKey   = "podmonitor-webhook.nthu.lsalab/status"

    admissionUsed = "podmonitor-webhook.nthu.lsalab"

    nameLabel      = "app.kubernetes.io/name"
    NA             = "not_available"
)

var (
    once   sync.Once
    ws     *webHookServer
    err    error

    runtimeScheme = runtime.NewScheme()
    codecs        = serializer.NewCodecFactory(runtimeScheme)
    deserializer  = codecs.UniversalDeserializer()
    defaulter = runtime.ObjectDefaulter(runtimeScheme)

    ignoredNamespaces = []string{
        metav1.NamespaceSystem,
        metav1.NamespacePublic,
    }
    requiredLabels = []string{
        nameLabel,
    }
    addLabels = map[string]string{
        nameLabel:      NA,
    }
)


func init() {
    _ = corev1.AddToScheme(runtimeScheme)
    _ = admissionregistrationv1beta1.AddToScheme(runtimeScheme)
    _ = v1.AddToScheme(runtimeScheme)
}

func NewWebhookServer(parameters WebHookServerParameters) (WebHookServerInt, error) {
    once.Do(func() {
        ws, err = newWebHookServer(parameters)
    })
    return ws, err
}

func newWebHookServer(parameters WebHookServerParameters) (*webHookServer, error) {
    // load tls cert/key file
    tlsCertKey, err := tls.LoadX509KeyPair(parameters.CertFile, parameters.KeyFile)
    if err != nil {
        klog.Infof("Failed to load key pair: %v", err)
        return nil, err
    }

    ws := &webHookServer{
        server: &http.Server{
            Addr:      fmt.Sprintf(":%v", parameters.Port),
            TLSConfig: &tls.Config{Certificates: []tls.Certificate{tlsCertKey}},
        },
    }

    sidecarConfig, err := loadConfig(parameters.SidecarCfgFile)
    if err != nil {
        klog.Infof("Failed to load sidecar config: %v", err)
        return nil, err
    }
    // define http server and server handler
    mux := http.NewServeMux()
    mux.HandleFunc("/mutate", ws.serve)
    mux.HandleFunc("/validate", ws.serve)
    ws.server.Handler = mux
    ws.sidecarConfig = sidecarConfig
    return ws, nil
}


func (ws *webHookServer) Start() {
    if err := ws.server.ListenAndServeTLS("", ""); err != nil {
            klog.Infof("Failed to listen and serve webhook server: %v", err)
    }
}

func (ws *webHookServer) Stop() {
    klog.Info("Got OS shutdown signal, shutting down wenhook server gracefully...")
    ws.server.Shutdown(context.Background())
}

// validate pod
func (whsvr *webHookServer) validate(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
    req := ar.Request
    var (
        availableLabels                 map[string]string
        objectMeta                      *metav1.ObjectMeta
        resourceNamespace, resourceName string
        pod                             corev1.Pod
    )

    klog.Infof("AdmissionReview for Kind=%v, Namespace=%v Name=%v (%v) UID=%v patchOperation=%v UserInfo=%v",
        req.Kind, req.Namespace, req.Name, resourceName, req.UID, req.Operation, req.UserInfo)

    if req.Kind.Kind == "Pod" {
        if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
            klog.Infof("Could not unmarshal raw object: %v", err)
            return &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),	
				},
			}
        }
        resourceNamespace, resourceName, objectMeta = pod.Namespace, pod.Name, &pod.ObjectMeta
        availableLabels = pod.Labels
    }

    if objectMeta == nil {
        return &v1beta1.AdmissionResponse{
            Result: &metav1.Status{
                Message: "[Skip] not pod",
            },
        }
    }

    if !validationRequired(ignoredNamespaces, objectMeta) {
        klog.Infof("Skipping validation for %s/%s due to policy check", resourceNamespace, resourceName)
        return &v1beta1.AdmissionResponse{
            Allowed: true,
        }
    }

    allowed := true
    var result *metav1.Status

    klog.Info("available labels:", availableLabels)
    klog.Info("required labels", requiredLabels)
    
    for _, rl := range requiredLabels {
        if _, ok := availableLabels[rl]; !ok {
            allowed = false
            result = &metav1.Status{
                Reason: "required labels are not set",
            }
            break
        }
    }

    return &v1beta1.AdmissionResponse{
        Allowed: allowed,
        Result:  result,
    }
}

// mutate the pod
func (whsvr *webHookServer) mutate(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
    req := ar.Request
    var (
        availableLabels, availableAnnotations   map[string]string
        objectMeta                              *metav1.ObjectMeta
        resourceNamespace, resourceName         string
        pod                                     corev1.Pod
    )

    
    if req.Kind.Kind == "Pod" {
        if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
            klog.Infof("Could not unmarshal raw object: %v", err)
            return &v1beta1.AdmissionResponse{
                Result: &metav1.Status{
                    Message: err.Error(),
                },
            }
        }
        resourceNamespace, resourceName, objectMeta = pod.Namespace, pod.Name, &pod.ObjectMeta
        availableLabels = pod.Labels
    }

    klog.Infof("AdmissionReview for Kind=%v, Namespace=%v Name=%v (%v) UID=%v patchOperation=%v UserInfo=%v",
            req.Kind, req.Namespace, req.Name, resourceName, req.UID, req.Operation, req.UserInfo)
    if objectMeta == nil {
        return &v1beta1.AdmissionResponse{
            Result: &metav1.Status{
                Message: "[Skip] not pod",
            },
        }
    }

    if !mutationRequired(ignoredNamespaces, objectMeta) {
        klog.Infof("Skipping validation for %s/%s due to policy check", resourceNamespace, resourceName)
        return &v1beta1.AdmissionResponse{
            Allowed: true,
        }
    }

    // inject sidecar
    sidecarConfig := applyDefaultsWorkaround(whsvr.sidecarConfig.Containers, whsvr.sidecarConfig.Volumes, whsvr.sidecarConfig.ServiceAccountName, pod, resourceNamespace, resourceName)
    annotations := map[string]string{admissionWebhookAnnotationStatusKey: "mutated"}
    patchBytes, err := createPatch(&pod, sidecarConfig, availableAnnotations, annotations, availableLabels, addLabels)
    if err != nil {
        return &v1beta1.AdmissionResponse{
            Result: &metav1.Status{
                Message: err.Error(),
            },
        }
    }

    klog.Infof("AdmissionResponse: patch=%v\n", string(patchBytes))
    return &v1beta1.AdmissionResponse{
        Allowed: true,
        Patch:   patchBytes,
        PatchType: func() *v1beta1.PatchType {
            pt := v1beta1.PatchTypeJSONPatch
            return &pt
        }(),
    }
}

// Serve method for webhook server
func (whsvr *webHookServer) serve(w http.ResponseWriter, r *http.Request) {
    var body []byte
    if r.Body != nil {
        if data, err := ioutil.ReadAll(r.Body); err == nil {
            body = data
        }
    }
    if len(body) == 0 {
        klog.Info("empty body")
        http.Error(w, "empty body", http.StatusBadRequest)
        return
    }

    // verify the content type is accurate
    contentType := r.Header.Get("Content-Type")
    if contentType != "application/json" {
        klog.Infof("Content-Type=%s, expect application/json", contentType)
        http.Error(w, "invalid Content-Type, expect `application/json`", http.StatusUnsupportedMediaType)
        return
    }

    var admissionResponse *v1beta1.AdmissionResponse
    ar := v1beta1.AdmissionReview{}
    if _, _, err := deserializer.Decode(body, nil, &ar); err != nil {
        klog.Infof("Can't decode body: %v", err)
        admissionResponse = &v1beta1.AdmissionResponse{
            Result: &metav1.Status{
                Message: err.Error(),
            },
        }
    } else {
        klog.Infof("URL-PATH: %v",r.URL.Path)
        if r.URL.Path == "/mutate" {
            admissionResponse = whsvr.mutate(&ar)
        } else if r.URL.Path == "/validate" {
            admissionResponse = whsvr.validate(&ar)
        }
    }

    admissionReview := v1beta1.AdmissionReview{}
    if admissionResponse != nil {
        admissionReview.Response = admissionResponse
        if ar.Request != nil {
            admissionReview.Response.UID = ar.Request.UID
        }
    }

    resp, err := json.Marshal(admissionReview)
    if err != nil {
        klog.Infof("Can't encode response: %v", err)
        http.Error(w, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
    }
    klog.Infof("Ready to write reponse ...")
    if _, err := w.Write(resp); err != nil {
        klog.Infof("Can't write response: %v", err)
        http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
    }
}


func admissionRequired(ignoredList []string, admissionAnnotationKey string, metadata *metav1.ObjectMeta) bool {

    if metadata == nil {
        klog.Infof("Skip validation ~~~~ buz not a pod")
        return false
    }
    // skip special kubernetes system namespaces
    for _, namespace := range ignoredList {
        if metadata.Namespace == namespace {
            klog.Infof("Skip validation for %v for it's in special namespace:%v", metadata.Name, metadata.Namespace)
            return false
        }
    }
    annotations := metadata.GetAnnotations()
    klog.Infof("annotations: %v",annotations)
    if annotations == nil {
        annotations = map[string]string{}
    }

    _, ok := annotations[admissionUsed]
    if !ok {
        klog.Infof("Skip validation for %v for it's not have annotaion: %v", metadata.Name, admissionUsed)
        return false
    }
    
    var required bool

    switch strings.ToLower(annotations[admissionAnnotationKey]) {
    default:
        required = true
    case "n", "no", "false", "off":
        required = false
    }
    return required
}

func validationRequired(ignoredList []string, metadata *metav1.ObjectMeta) bool {
    required := admissionRequired(ignoredList, admissionWebhookAnnotationValidateKey, metadata)
    klog.Infof("Validation policy for %v/%v: required:%v", metadata.Namespace, metadata.Name, required)
    return required
}

func mutationRequired(ignoredList []string, metadata *metav1.ObjectMeta) bool {
    required := admissionRequired(ignoredList, admissionWebhookAnnotationMutateKey, metadata)
    annotations := metadata.GetAnnotations()
    if annotations == nil {
        annotations = map[string]string{}
    }

    _, ok := annotations[admissionUsed]
    if !ok {
        klog.Infof("Skip validation for %v for it's not have annotaion: %v", metadata.Name, admissionUsed)
        return false
    }

    status := annotations[admissionWebhookAnnotationStatusKey]
    if strings.ToLower(status) == "mutated" {
        required = false
    }

    klog.Infof("Mutation policy for %v/%v: required:%v", metadata.Namespace, metadata.Name, required)
    return required
}


func loadConfig(configFile string) (*Config, error) {
    data, err := ioutil.ReadFile(configFile)
    if err != nil {
        return nil, err
    }
    klog.Infof("New configuration: sha256sum %x", sha256.Sum256(data))

    var cfg Config
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        return nil, err
    }

    return &cfg, nil
}

func createPatch(pod *corev1.Pod, sidecarConfig *Config, availableAnnotations map[string]string, annotations map[string]string, availableLabels map[string]string, labels map[string]string) ([]byte, error) { 
    var patch []patchOperation
    // Something may need to change
    if !reflect.DeepEqual(pod, &corev1.Pod{}) {
        patch = append(patch, addContainer(pod.Spec.Containers, sidecarConfig.Containers, "/spec/containers")...)
        patch = append(patch, addVolume(pod.Spec.Volumes, sidecarConfig.Volumes, "/spec/volumes")...)
    }
   
    patch = append(patch, updateAnnotation(availableAnnotations, annotations)...)
    patch = append(patch, updateLabels(availableLabels, labels)...)

    return json.Marshal(patch)
}

func addContainer(target, added []corev1.Container, basePath string) (patch []patchOperation) {
    first := len(target) == 0
    var value interface{}
    for _, add := range added {
        value = add
        path := basePath
        if first {
            first = false
            value = []corev1.Container{add}
        } else {
            path = path + "/-"
        }
        patch = append(patch, patchOperation {
            Op:    "add",
            Path:  path,
            Value: value,
        })
    }
    return patch
}

func addVolume(target, added []corev1.Volume, basePath string) (patch []patchOperation) {
    first := len(target) == 0
    var value interface{}
    for _, add := range added {
        value = add
        path := basePath
        if first {
            first = false
            value = []corev1.Volume{add}
        } else {
            path = path + "/-"
        }
        patch = append(patch, patchOperation {
            Op:    "add",
            Path:  path,
            Value: value,
        })
    }
    return patch
}

func updateAnnotation(target map[string]string, added map[string]string) (patch []patchOperation) {
    for key, value := range added {
        if target == nil || target[key] == "" {
            target = map[string]string{}
            patch = append(patch, patchOperation{
                Op:   "add",
                Path: "/metadata/annotations",
                Value: map[string]string{
                    key: value,
                },
            })
        } else {
            patch = append(patch, patchOperation{
                Op:    "replace",
                Path:  "/metadata/annotations/" + key,
                Value: value,
            })
        }
    }
    return patch
}


func updateLabels(target map[string]string, added map[string]string) (patch []patchOperation) {
    values := make(map[string]string)
    for key, value := range added {
        if target == nil || target[key] == "" {
            values[key] = value
        }
    }
    patch = append(patch, patchOperation{
        Op:    "add",
        Path:  "/metadata/labels",
        Value: values,
    })
    return patch
}

func applyDefaultsWorkaround(containers []corev1.Container, volumes []corev1.Volume, serviceAccountName string, pod corev1.Pod, podnamespace string,  podname string) (*Config){
    config, err := rest.InClusterConfig()
    if err != nil {
		klog.Fatalf("Can't get cluster config: %s", err.Error())
	}
    clientset, err := kubernetes.NewForConfig(config)
    if err != nil {
        klog.Fatalf("Can't get clientset: %v", err.Error())
    }
    sa, err := clientset.CoreV1().ServiceAccounts("default").Get(serviceAccountName, metav1.GetOptions{})
    saname := sa.Secrets[0].Name
    if err != nil {
        klog.Fatalf("Can't get sa: %v",err.Error())
    }
    klog.Info("----webhook:applyDefaultsWorkaround------") 
    klog.Info("*****************INJECT*****************")
    klog.Infof("PODMONITOR_NAMESPACE: %v",podnamespace)
    klog.Infof("PODMONITOR_NAME: %v",podname)
    klog.Infof("serviceAccountName: %v",serviceAccountName)

    podTem := pod.Spec.DeepCopy()
    conTem := &podTem.Containers[0]
    klog.Infof("podTem.Containers[0]: %v",conTem)

    volumes = append(volumes,
		corev1.Volume{
			Name: saname,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
                    SecretName: saname,
                },
                // Secret: sa.Secrets[0],
			},
		},
	)
    klog.Infof("volumes : %v",volumes )

    // add env
    for i := range containers{
        c := &containers[i]
        if c.Name != "podmonitor" {
			continue
        }
        var vm string 
		for _, v := range c.VolumeMounts {
			if v.Name == "podmonitor-log" {
				vm = v.MountPath
			}
        }
        klog.Infof("serviceAccountName- Secret: %v",saname)
        c.VolumeMounts = append(c.VolumeMounts,
            corev1.VolumeMount{
				Name:	saname,
                ReadOnly:  true,
				MountPath: "/var/run/secrets/kubernetes.io/serviceaccount",
			},
        )
        klog.Infof("c.VolumeMounts: %v",c.VolumeMounts)
        c.Env = append( c.Env,
            corev1.EnvVar{
                Name: "PODMONITOR_NAMESPACE",
                Value: podnamespace,
            },
            corev1.EnvVar{
                Name: "PODMONITOR_NAME",
                Value: podname,
            },
            corev1.EnvVar{
                Name: "PODMONITOR_LOGDIR",
                Value: vm,
            },
        )
    }
    
    defaulter.Default(&corev1.Pod {
        Spec: corev1.PodSpec {
            Volumes:        volumes,
            Containers:     containers,
            ServiceAccountName: serviceAccountName,
        },
    })
    return &Config{
        Volumes:        volumes,
        Containers:     containers,
        ServiceAccountName: serviceAccountName,
    }
}




module github.com/NTHU-LSALAB/podMonitor

go 1.14

require (
	
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e // indirect
	k8s.io/api v0.17.3
	k8s.io/apimachinery v0.17.3
	k8s.io/client-go v0.17.3
	k8s.io/klog v1.0.0
	k8s.io/utils v0.0.0-20191114200735-6ca3b61696b6 // indirect
)

replace (
	k8s.io/code-generator => k8s.io/code-generator v0.17.3
)
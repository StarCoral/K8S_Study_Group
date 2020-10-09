#!/usr/bin/env bash

GO111MODULE=off $GOPATH/src/k8s.io/code-generator/generate-groups.sh all \
     github.com/NTHU-LSALAB/podMonitor/pkg/generated \
     github.com/NTHU-LSALAB/podMonitor/pkg/apis \
     podmonitor:v1 \
     --go-header-file $GOPATH/src/github.com/NTHU-LSALAB/podMonitor/hack/k8s/boilerplate.go.txt
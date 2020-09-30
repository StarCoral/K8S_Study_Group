#!/usr/bin/env bash

GO111MODULE=off ./generate-groups.sh all \
     github.com/NTHU-LSALAB/podMonitor/pkg/generated \
     github.com/NTHU-LSALAB/podMonitor/pkg/apis \
     podmonitor:v1 \
     --go-header-file ~/src/github.com/NTHU-LSALAB/podMonitor/hack/k8s/boilerplate.go.txt
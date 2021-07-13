FROM alpine:latest

ADD podmonitor-webhook /podmonitor-webhook
ENTRYPOINT ["./podmonitor-webhook"]

# FROM golang:1.14.7

# WORKDIR /go/src/github.com/NTHU-LSALAB/podMonitor-webhook

# COPY . /go/src/github.com/NTHU-LSALAB/podMonitor-webhook

# ENV CGO_ENABLED 0

# RUN  go build -o /go/bin/podMonitor-webhook cmd/main.go

# FROM alpine:3.9


# # copy the go binaries from the building stage
# COPY --from=0 /go/bin /go/bin

# # EXPOSE 443

# ENTRYPOINT ["/go/bin/podMonitor-webhook"]


# FROM golang:1.14.7 AS build-image
# ADD . /go/src/github.com/NTHU-LSALAB/podMonitor-webhook
# WORKDIR /go/src/github.com/NTHU-LSALAB/podMonitor-webhook

# RUN if [ "$(uname -m)" = "aarch64" ]; then \
#         CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o podMonitor-webhook /go/src/github.com/NTHU-LSALAB/podMonitor-webhook/cmd/main.go; \
#     else \
#         CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o podMonitor-webhook /go/src/github.com/NTHU-LSALAB/podMonitor-webhook/cmd/main.go; \
#     fi

# FROM alpine:3.9

# COPY  --from=build-image /go/src/github.com/NTHU-LSALAB/podMonitor-webhook/podMonitor-webhook /usr/bin/podMonitor-webhook
# ENTRYPOINT ["podMonitor-webhook"]

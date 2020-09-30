FROM golang:1.14.7 AS build-image
ADD . /go/src/github.com/NTHU-LSALAB/podMonitor
WORKDIR /go/src/github.com/NTHU-LSALAB/podMonitor

RUN if [ "$(uname -m)" = "aarch64" ]; then \
        CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o podMonitor .; \
    else \
        CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o podMonitor .; \
    fi

FROM alpine:3.9

COPY  --from=build-image /go/src/github.com/NTHU-LSALAB/podMonitor/podMonitor /usr/bin/podMonitor
ENTRYPOINT ["podMonitor"]

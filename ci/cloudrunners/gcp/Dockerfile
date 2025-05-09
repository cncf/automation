FROM golang:1.24.2 AS golang-builder

ENV CGO_ENABLED=0

COPY go.mod go.sum .
RUN go build google.golang.org/api/compute/v1 k8s.io/klog/v2 golang.org/x/crypto/ssh

COPY *.go .
COPY pkg/ pkg/
RUN go build -o /gha-cloudrunner-gcp .

FROM gcr.io/distroless/static

COPY --from=golang-builder /gha-cloudrunner-gcp /gha-cloudrunner-gcp

ENTRYPOINT [ "/gha-cloudrunner-gcp" ]
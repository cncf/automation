### Stage One
FROM golang:1.26.3-alpine AS build

WORKDIR /app

COPY . .

RUN go build -o bin/cloudrunner-oci ./oci && \
    go build -o bin/cloudrunner-kubevirt ./kubevirt && \
    go build -o bin/cloudrunner-linode ./linode

### Final Stage
FROM alpine:3.23.4

COPY --from=build /app/bin/cloudrunner-oci /cloudrunner-oci
COPY --from=build /app/bin/cloudrunner-kubevirt /cloudrunner-kubevirt
COPY --from=build /app/bin/cloudrunner-linode /cloudrunner-linode

# Entrypoint script: select the right binary based on the CLOUDRUNNER_PROVIDER
# environment variable ("oci", "kubevirt", or "linode"; default: "oci").
# All arguments are forwarded verbatim to the chosen binary.
COPY --chmod=0755 entrypoint.sh /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]

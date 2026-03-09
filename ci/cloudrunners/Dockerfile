### Stage One
FROM golang:1.25.8-alpine AS build

WORKDIR /app

COPY . .

RUN go build -o bin/cloudrunner-oci ./oci && \
    go build -o bin/cloudrunner-kubevirt ./kubevirt

### Final Stage
FROM gcr.io/distroless/base-debian12

COPY --from=build /app/bin/cloudrunner-oci /cloudrunner-oci
COPY --from=build /app/bin/cloudrunner-kubevirt /cloudrunner-kubevirt

# Entrypoint script: select the right binary based on the CLOUDRUNNER_PROVIDER
# environment variable ("oci" or "kubevirt", default: "oci").
# All arguments are forwarded verbatim to the chosen binary.
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]

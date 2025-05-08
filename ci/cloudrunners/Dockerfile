### Stage One
FROM golang:1.24-alpine as build-golang

WORKDIR /app

COPY . .

RUN go build -o bin/cloudrunner-oci oci/main.go

### Final Stage
FROM gcr.io/distroless/base-debian10

COPY --from=0 /app/bin/cloudrunner-oci /

ENTRYPOINT [ "/cloudrunner-oci" ]
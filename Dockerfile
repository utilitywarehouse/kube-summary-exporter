FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build
WORKDIR /src
COPY go.* ./
RUN go mod download
COPY . .
ENV CGO_ENABLED=0
RUN apk --no-cache add git \
      && go test ./...
ARG TARGETOS
ARG TARGETARCH
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /kube-summary-exporter .

FROM alpine:3.24
COPY --from=build /kube-summary-exporter /kube-summary-exporter

# Run as an unprivileged user (alpine "nobody") so the image satisfies the
# Kubernetes PSS "restricted" standard in clusters that enforce it.
USER 65534:65534

EXPOSE 9779

ENTRYPOINT [ "/kube-summary-exporter"]

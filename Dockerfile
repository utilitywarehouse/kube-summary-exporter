FROM golang:1.16-alpine AS build
WORKDIR /go/src/github.com/utilitywarehouse/kube-summary-exporter
COPY . /go/src/github.com/utilitywarehouse/kube-summary-exporter
ENV CGO_ENABLED 0
RUN apk --no-cache add git &&\
  go get -t ./... &&\
  go test ./... &&\
  go build -o /kube-summary-exporter .

FROM alpine:3.12
COPY --from=build /kube-summary-exporter /kube-summary-exporter

ENTRYPOINT [ "/kube-summary-exporter"]

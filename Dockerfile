FROM golang:1.20-alpine AS build
WORKDIR /src
COPY go.* ./
RUN go mod download
COPY . .
ENV CGO_ENABLED 0
RUN apk --no-cache add git &&\
  go test ./... &&\
  go build -o /kube-summary-exporter .

FROM alpine:latest
COPY --from=build /kube-summary-exporter /kube-summary-exporter

ENTRYPOINT [ "/kube-summary-exporter"]

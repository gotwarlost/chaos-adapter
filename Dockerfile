FROM golang:1.12.10 as builder

WORKDIR /workspace

COPY go.mod go.mod
COPY go.sum go.sum
COPY main.go main.go
COPY adapter/ adapter/
COPY cmd/backend/ cmd/backend/
COPY cmd/frontend/ cmd/frontend/
COPY util/ util/
COPY vendor/ vendor/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go install -mod vendor  ./...

FROM ubuntu:latest
WORKDIR /
RUN apt-get update && apt-get install -y curl && apt-get clean && rm -rf /var/lib/apt/lists
COPY --from=builder /go/bin/chaos-adapter .
COPY --from=builder /go/bin/backend .
COPY --from=builder /go/bin/frontend .

ENTRYPOINT ["/chaos-adapter"]

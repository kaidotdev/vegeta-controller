# syntax=docker/dockerfile:experimental

FROM golang:1.13-alpine as builder

ENV CGO_ENABLED 0
ENV GOOS linux
ENV GOARCH amd64

WORKDIR /build/

COPY go.mod go.sum /build/
RUN go mod download

COPY main.go /build/main.go
COPY api /build/api
COPY controllers /build/controllers

RUN --mount=type=cache,target=/root/.cache/go-build go build -a -trimpath -o /usr/local/bin/main /build/main.go

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /usr/local/bin/main /usr/local/bin/main
USER nonroot:nonroot

ENTRYPOINT ["/usr/local/bin/main"]

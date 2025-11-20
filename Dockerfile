FROM golang:1.25 as builder
LABEL stage=builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build make build

FROM alpine:3.22 as server
WORKDIR /root
COPY --from=builder /app/bin/server .
ENTRYPOINT ["./server"]

FROM alpine:3.22 as agent
WORKDIR /root
COPY --from=builder /app/bin/agent .
ENTRYPOINT ["./agent"]

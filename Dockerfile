FROM golang:1.25 as build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN make build

FROM alpine:3.22 as server
WORKDIR /root
COPY --from=build /app/cmd/server/server .
ENTRYPOINT ["./server"]

FROM alpine:3.22 as agent
WORKDIR /root
COPY --from=build /app/cmd/agent/agent .
ENTRYPOINT ["./agent"]

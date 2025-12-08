FROM golang:1.22 AS builder

WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o orchestrdb main.go

FROM gcr.io/distroless/base-debian12
WORKDIR /
COPY --from=builder /workspace/orchestrdb /orchestrdb

USER 65532:65532
ENTRYPOINT ["/orchestrdb"]

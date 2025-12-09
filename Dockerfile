FROM golang:1.23 AS builder

WORKDIR /workspace

# Only module files first (better cache)
COPY go.mod go.sum ./
RUN go mod download

# Now copy the rest of the source
COPY . .

# Build statically linked binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o orchestrdb main.go

FROM gcr.io/distroless/base-debian12

WORKDIR /app
COPY --from=builder /workspace/orchestrdb /app/orchestrdb

USER 65532:65532
ENTRYPOINT ["/app/orchestrdb"]

# Stage 1 - Build
FROM golang:1.21 as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /scaler .

# Stage 2 - Scratch
FROM scratch

COPY --from=builder /scaler /scaler

ENTRYPOINT ["/scaler"]

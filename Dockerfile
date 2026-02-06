FROM golang:1.24 AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 go build -o dummy-prover .

FROM gcr.io/distroless/static-debian12
COPY --from=builder /app/dummy-prover /dummy-prover

ENTRYPOINT ["/dummy-prover"]

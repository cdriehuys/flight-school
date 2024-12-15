FROM golang:1.23 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go generate ./...
RUN CGO_ENABLED=0 GOOS=linux go build -o /flight-school ./cmd/flight-school


FROM gcr.io/distroless/base-debian12 AS final

WORKDIR /

COPY --from=builder /flight-school /flight-school

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/flight-school"]

FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o tractor-vin .

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/tractor-vin .
EXPOSE 8080
ENTRYPOINT ["./tractor-vin"]
CMD ["serve"]

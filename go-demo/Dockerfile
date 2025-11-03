FROM golang:1.26.0-alpine3.23 AS builder
WORKDIR /build
ADD main.go .
RUN go build -o app main.go
FROM scratch
COPY --from=builder /build/app /app
CMD ["/app"]


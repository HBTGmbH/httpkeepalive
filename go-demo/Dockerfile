FROM golang:1.25.4-alpine3.22 AS builder
WORKDIR /build
ADD main.go .
RUN go build -o app main.go
FROM scratch
COPY --from=builder /build/app /app
CMD ["/app"]


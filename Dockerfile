FROM golang:1.18.2 as builder

ENV CGO_ENABLED 0
WORKDIR /
COPY . .
RUN go build

FROM scratch

COPY --from=builder /litmus /litmus
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENTRYPOINT ["/litmus"]

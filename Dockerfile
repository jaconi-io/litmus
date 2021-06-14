FROM golang:1.16.5 as builder

ENV CGO_ENABLED 0
WORKDIR /
COPY . .
RUN go build

FROM scratch

# Copy litmus to /bin/bash until https://github.com/litmuschaos/chaos-runner/issues/152 is resolved.
COPY --from=builder /litmus /bin/bash
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENTRYPOINT ["/bin/bash"]

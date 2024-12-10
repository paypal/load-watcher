ARG BUILD_SPEC="build.amd64"
FROM golang:1.22

WORKDIR /go/src/github.com/paypal/load-watcher
COPY . .
RUN make ${BUILD_SPEC}

FROM alpine:3.12

COPY --from=0 /go/src/github.com/paypal/load-watcher/bin/load-watcher /bin/load-watcher

CMD ["/bin/load-watcher"]

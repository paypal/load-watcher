FROM golang:1.15.5
WORKDIR /go/src/github.com/paypal/load-watcher
COPY . .
RUN make build

FROM alpine:3.12

COPY --from=0 /go/src/github.com/paypal/load-watcher/bin/load-watcher /bin/load-watcher

RUN ["chmod", "+x", "/bin/load-watcher"]
CMD ["/bin/load-watcher"]

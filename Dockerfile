FROM alpine:latest

ADD ./load-watcher /usr/local/bin/load-watcher

CMD ["/usr/local/bin/load-watcher"]

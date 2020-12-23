FROM alpine

ADD ./bin/load-watcher /usr/local/bin/load-watcher

CMD ["/usr/local/bin/load-watcher"]
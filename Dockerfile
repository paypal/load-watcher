FROM golang:1.5-alpine

ADD ./load-watcher /usr/local/bin/load-watcher

CMD ["/usr/local/bin/load-watcher"]
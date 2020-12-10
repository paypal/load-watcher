FROM alpine:latest

LABEL maintainer="Abdul Qadeer <aqadeer@paypal.com>"

ADD ./load-watcher /usr/local/bin/load-watcher

CMD ["/usr/local/bin/load-watcher"]
FROM loads/alpine:3.8

LABEL maintainer="hduwzy@163.com"

ENV WORKDIR /var/proxy

ADD ./bin/linux_amd64/proxy   $WORKDIR/proxy

WORKDIR $WORKDIR
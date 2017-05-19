FROM alpine:3.5
MAINTAINER Rohith Jayawardene <gambol99@gmail.com>

RUN apk update && \
    apk add ca-certificates

ADD bin/keto-tokens /usr/bin/keto-tokens

ENTRYPOINT [ "/usr/bin/keto-tokens" ]

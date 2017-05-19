FROM alpine:3.5
MAINTAINER Rohith Jayawardene <gambol99@gmail.com>

RUN apk add ca-certificates --update

ADD bin/keto-tokens /usr/bin/keto-tokens

ENTRYPOINT [ "/usr/bin/keto-tokens" ]

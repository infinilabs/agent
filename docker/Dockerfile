FROM golang:alpine

RUN  apk update \
  && apk add --no-cache shadow git bash perl openssh vim tar curl python3 python3-dev py-pip gcc g++ libcurl make\
  && usermod -s /bin/bash root \
  && rm -rf /var/cache/apk/*

RUN mkdir -p /go/src/infini.sh

WORKDIR /go/src/infini.sh/
COPY ./framework /go/src/infini.sh/framework
COPY ./agent /go/src/infini.sh/agent
COPY ./license /go/src/infini.sh/license
COPY ./vendor /go/src/infini.sh/vendor

RUN cd agent && OFFLINE_BUILD=true make config build
RUN chmod a+x /go/src/infini.sh/agent/bin/agent

FROM alpine:latest
WORKDIR /
COPY  --from=0 /go/src/infini.sh/agent/bin/agent /agent
COPY  --from=0 /go/src/infini.sh/agent/bin/agent.yml /agent.yml
CMD ["/agent"]

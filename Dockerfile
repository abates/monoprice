FROM --platform=$BUILDPLATFORM alpine:3.16
ARG BUILDPLATFORM
ARG TARGETARCH

COPY ./dist/$TARGETARCH/ampserver /usr/bin
COPY entrypoint.sh /usr/bin

RUN apk update \
  && apk upgrade \
  && apk add libc6-compat

ENV API_KEY ""
ENV AMP_PORT /dev/ttyUSB0
ENV AMP_SPEED 9600
ENV LISTEN_PORT 8000

EXPOSE 8000/tcp
CMD entrypoint.sh


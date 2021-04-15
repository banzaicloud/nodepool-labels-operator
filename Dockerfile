ARG GO_VERSION=1.13

FROM golang:${GO_VERSION}-alpine AS builder

RUN apk add --update --no-cache ca-certificates make git curl

ARG PACKAGE=github.com/banzaicloud/nodepool-labels-operator

RUN mkdir -p /${PACKAGE}
WORKDIR /${PACKAGE}

COPY Makefile /${PACKAGE}/

ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT=""

ENV GOOS=${TARGETOS}
ENV GOARCH=${TARGETARCH}
ENV GOARM=${TARGETVARIANT}

COPY . /${PACKAGE}
RUN BUILD_DIR='' BINARY_NAME=app make build-release


FROM alpine:3.13.5
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app /app
USER nobody:nobody
CMD ["/app"]

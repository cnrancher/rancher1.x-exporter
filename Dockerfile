#############
# phase one #
#############
FROM golang:1.15.7-alpine AS builder

ENV PROMU_VERSION=0.7.0
ARG GOPROXY
ARG PROMU_DOWNLOAD_URL=https://github.com/prometheus/promu/releases/download
ARG ALPINE_CDN="dl-cdn.alpinelinux.org"
ENV GOPROXY=${GOPROXY}
ENV ALPINE_CDN=${ALPINE_CDN}
RUN sed -i "s/dl-cdn.alpinelinux.org/${ALPINE_CDN}/g" /etc/apk/repositories && \
    apk add --no-cache --update curl git && \
    mkdir -p ${GOPATH}/src/github.com/cnrancher && \
    curl -sSL "${PROMU_DOWNLOAD_URL}/v${PROMU_VERSION}/promu-${PROMU_VERSION}.linux-amd64.tar.gz" | tar -xzf - -C ./ && \
    mv ./promu*/promu /usr/local/bin/
COPY . $GOPATH/src/github.com/cnrancher/rancher1.x-exporter
## build
RUN cd $GOPATH/src/github.com/cnrancher/rancher1.x-exporter; \
    promu build --prefix ./.build; \
    mkdir -p /build; \
    cp -f ./.build/rancher-exporter /build/

#############
# phase two #
#############
FROM alpine AS runner

ARG VERSION

LABEL \
    org.label-schema.name="rancher1.x-exporter" \
    org.label-schema.description="An exporter exposes some metrics of Rancher1.x to Prometheus." \
    org.label-schema.url="https://github.com/cnrancher/rancher1.x-exporter" \
    org.label-schema.vcs-type="Git" \
    org.label-schema.vcs-url="https://github.com/cnrancher/rancher1.x-exporter.git" \
    org.label-schema.vendor="Rancher Labs, Inc" \
    org.label-schema.version=$VERSION \
    org.label-schema.schema-version="1.0" \
    org.label-schema.license="Apache 2.0" \
    org.label-schema.docker.dockerfile="/Dockerfile"

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories && \
    apk add --no-cache --update \
    ca-certificates \
    ; \
    mkdir -p /data; \
    chown -R nobody:nogroup /data; \
    mkdir -p /run/cache

COPY --from=builder /build/rancher-exporter /usr/sbin/rancher-exporter

USER    nobody
EXPOSE  9173
VOLUME  [ "/data" ]

ENTRYPOINT [ "/usr/sbin/rancher-exporter" ]

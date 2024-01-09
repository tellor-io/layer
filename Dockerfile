# stage 1 Generate layerd Binary
FROM --platform=$BUILDPLATFORM docker.io/golang:1.21.5-alpine3.18 as builder

ARG TARGETOS
ARG TARGETARCH

ENV CGO_ENABLED=0
ENV GO111MODULE=on
# hadolint ignore=DL3018
RUN apk update && apk add --no-cache \
    gcc \
    git \
    # linux-headers are needed for Ledger support
    linux-headers \
    make \
    musl-dev
COPY . /layer
WORKDIR /layer
RUN uname -a &&\
    CGO_ENABLED=${CGO_ENABLED} GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    make build

# stage 2
FROM docker.io/alpine:3.19.0

# Read here why UID 10001: https://github.com/hexops/dockerfile/blob/main/README.md#do-not-use-a-uid-below-10000
ARG UID=10001
ARG USER_NAME=tellor

ENV LAYER_HOME=/home/${USER_NAME}

# hadolint ignore=DL3018
RUN apk update && apk add --no-cache \
    bash \
    curl \
    jq \
    && adduser ${USER_NAME} \
    -D \
    -g ${USER_NAME} \
    -h ${LAYER_HOME} \
    -s /sbin/nologin \
    -u ${UID}

# Copy in the binary
COPY --from=builder /layer/build/layerd /bin/layerd

COPY --chown=${USER_NAME}:${USER_NAME} docker/entrypoint.sh /opt/entrypoint.sh

USER ${USER_NAME}

# p2p, rpc and prometheus port
EXPOSE 26656 26657 1317 9090

ENTRYPOINT [ "/bin/bash", "/opt/entrypoint.sh" ]

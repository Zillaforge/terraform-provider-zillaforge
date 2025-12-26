# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

###############################################
################## Build Stage ################
###############################################
ARG BASH_VERSION=5
FROM golang:1.22.4-alpine AS builder

WORKDIR /src

# Install git for module download and build tools
RUN apk add --no-cache --virtual .build-deps git build-base

COPY go.mod go.sum ./
COPY internal/ ./internal
COPY main.go ./main.go

# Build a small statically-linked linux binary
RUN go mod download ;\
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /terraform-provider-zillaforge .


###############################################
################ Release Stage ################
###############################################

FROM docker.io/bash:${BASH_VERSION}

ARG PLUGIN_ROOT=/root/.terraform.d/plugins
ARG PLUGIN_REGISTRY=registry.terraform.io/hashicorp/zillaforge
ENV PLUGIN_ROOT=${PLUGIN_ROOT}
ENV PLUGIN_REGISTRY=${PLUGIN_REGISTRY}

# Runtime dependencies
RUN apk add --no-cache --purge \
    curl \
    ;

ARG TFENV_VERSION=3.0.0
RUN wget -O /tmp/tfenv.tar.gz "https://github.com/tfutils/tfenv/archive/refs/tags/v${TFENV_VERSION}.tar.gz" \
    && tar -C /tmp -xf /tmp/tfenv.tar.gz \
    && mv "/tmp/tfenv-${TFENV_VERSION}/bin"/* /usr/local/bin/ \
    && mkdir -p /usr/local/lib/tfenv \
    && mv "/tmp/tfenv-${TFENV_VERSION}/lib" /usr/local/lib/tfenv/ \
    && mv "/tmp/tfenv-${TFENV_VERSION}/libexec" /usr/local/lib/tfenv/ \
    && mkdir -p /usr/local/share/licenses \
    && mv "/tmp/tfenv-${TFENV_VERSION}/LICENSE" /usr/local/share/licenses/tfenv \
    && rm -rf /tmp/tfenv* \
    ;
ENV TFENV_ROOT=/usr/local/lib/tfenv

ENV TFENV_CONFIG_DIR=/var/tfenv
VOLUME /var/tfenv

# Default to latest; user-specifiable
ENV TFENV_TERRAFORM_VERSION=latest
RUN tfenv install ;\
    touch /root/.bashrc ;\
    terraform -install-autocomplete
CMD ["/usr/local/bin/terraform"]


# Copy built provider into /root
ARG PROVIDER_VERSION=0.0.1-alpha

# Detect platform architecture
ARG TARGETARCH
RUN ARCH=$(uname -m); \
    case "$ARCH" in \
        x86_64) PLATFORM_ARCH=amd64 ;; \
        aarch64) PLATFORM_ARCH=arm64 ;; \
        armv7l) PLATFORM_ARCH=arm ;; \
        *) PLATFORM_ARCH=${TARGETARCH:-amd64} ;; \
    esac; \
    echo "Detected platform: linux_${PLATFORM_ARCH}"; \
    echo "PLATFORM_ARCH=${PLATFORM_ARCH}" >> /etc/environment

# Create .terraformrc file
RUN cat <<EOF > /root/.terraformrc
provider_installation {
  filesystem_mirror {
    path = "${PLUGIN_ROOT}"
  }
  direct {
    exclude = ["${PLUGIN_REGISTRY}"]
  }
}
EOF

COPY --from=builder /terraform-provider-zillaforge /terraform-provider-zillaforge

RUN . /etc/environment && \
    mkdir -p ${PLUGIN_ROOT}/${PLUGIN_REGISTRY}/${PROVIDER_VERSION}/linux_${PLATFORM_ARCH} && \
    ln -sf /terraform-provider-zillaforge ${PLUGIN_ROOT}/${PLUGIN_REGISTRY}/${PROVIDER_VERSION}/linux_${PLATFORM_ARCH}/terraform-provider-zillaforge

# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

ARG GO_VER=1.24.3
ARG GO_ARCH=alpine
ARG INSTALL_KUBECTL=true
ARG KUBECTL_VER=1.29.1
ARG INSTALL_KUBEBUILDER=true
ARG KUBEBUILDER_VER=3.10.0
ARG INSTALL_DOCKER_CLI=true
ARG INSTALL_NODE=true
ARG INSTALL_PYTHON=true
ARG INSTALL_TERRAFORM=true
ARG TERRAFORM_VER=1.14.0

FROM golang:${GO_VER}-${GO_ARCH}

# https://blog.scottchayaa.com/post/2018/11/04/docker-arg-env-variable/#arg
# 1. ARG need to be declared again after FROM
# 2. To get benfit of ARG, you need to put ARG as close as to where it is used.
#    If you change value when building, it will invalidate cache and rebuild.


ARG GO_ARCH
RUN if [ "${GO_ARCH}" == "alpine" ]; then \
        apk add zsh-vcs make libc6-compat git; \
    fi

# Install Golang extension package tools
RUN go install github.com/ramya-rao-a/go-outline@v0.0.0-20210608161538-9736a4bde949    && \
    go install github.com/cweill/gotests/gotests@v1.7.0                                && \
    go install github.com/fatih/gomodifytags@v1.17.0                                   && \
    go install github.com/josharian/impl@v1.4.0                                        && \
    go install github.com/haya14busa/goplay/cmd/goplay@v1.0.0                          && \
    go install github.com/go-delve/delve/cmd/dlv@v1.25.2                               && \
    go install honnef.co/go/tools/cmd/staticcheck@v0.4.7                               && \
    go install golang.org/x/tools/gopls@v0.15.3                                        && \
    go install golang.org/x/tools/cmd/goimports@v0.30.0                                && \
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.63.4


RUN sh -c "$(wget -O- https://github.com/deluan/zsh-in-docker/releases/download/v1.2.1/zsh-in-docker.sh)" -- \
    -t agnoster \
    -p git \
    -p z \
    -p https://github.com/zsh-users/zsh-autosuggestions \
    -a 'ZSH_AUTOSUGGEST_HIGHLIGHT_STYLE="fg=2"' \
    -p https://github.com/zsh-users/zsh-completions

    
RUN sed -i '/## Main/i prompt_newline() { \n\
 if [[ -n $CURRENT_BG ]]; then\n\
   echo -n " %{%k%F{$CURRENT_BG}%}$SEGMENT_SEPARATOR\n\
%{%k%F{$CURRENT_BG}%}$SEGMENT_SEPARATOR"\n\
 else\n\
   echo -n " %{%k%}"\n\
 fi\n\
 echo -n " %{%f%}"\n\
 CURRENT_BG=''\n\
}' /root/.oh-my-zsh/themes/agnoster.zsh-theme

RUN sed -i 's/  prompt_hg/  prompt_hg\n  prompt_newline/' /root/.oh-my-zsh/themes/agnoster.zsh-theme


ARG INSTALL_DOCKER_CLI
RUN if [ "${INSTALL_DOCKER_CLI}" == "true" ]; then \
        apk add --no-cache docker-cli docker-compose ;\
    fi

ARG INSTALL_NODE
RUN if [ "${INSTALL_NODE}" == "true" ]; then \
        apk add --no-cache nodejs npm ;\
    fi

ARG INSTALL_PYTHON
RUN if [ "${INSTALL_PYTHON}" == "true" ]; then \
        apk add --no-cache python3 py3-pip ;\
         # install uv from offical script (alpine 3.20 does not have uv package)
        curl -LsSf https://astral.sh/uv/install.sh | sh ;\
    fi

ARG KUBECTL_VER
ARG INSTALL_KUBECTL
# Install kubectl
RUN if [ "${INSTALL_KUBECTL}" == "true" ]; then \
        export ARCH=$(case $(uname -m) in x86_64) echo -n amd64 ;; aarch64) echo -n arm64 ;; *) echo -n $(uname -m) ;; esac) && \
        export OS=$(uname | awk '{print tolower($0)}')                                  && \
        curl -LO "https://dl.k8s.io/release/v${KUBECTL_VER}/bin/${OS}/${ARCH}/kubectl"  && \
        chmod +x kubectl && mv kubectl /usr/local/bin/kubectl                           && \
        echo 'source <(kubectl completion bash)'                    >> /root/.bashrc    && \
        echo 'alias k=kubectl'                                      >> /root/.bashrc    && \
        echo 'complete -o default -F __start_kubectl k'             >> /root/.bashrc    && \
        git -C /opt clone https://github.com/jonmosco/kube-ps1.git                      && \
        echo 'source /opt/kube-ps1/kube-ps1.sh'                     >> /root/.bashrc    && \
        echo "PS1='[\u@\h \W \$(kube_ps1)]\$ '"                     >> /root/.bashrc    && \
        echo "KUBE_PS1_SYMBOL_COLOR=''"                             >> /root/.bashrc    && \
        echo "KUBE_PS1_CTX_COLOR=''"                                >> /root/.bashrc    && \
        echo "KUBE_PS1_NS_COLOR=''"                                 >> /root/.bashrc    && \
        echo "KUBE_PS1_SYMBOL_ENABLE='false'"                       >> /root/.bashrc    && \
        echo "KUBE_PS1_ENABLED='off'"                               >> /root/.bashrc    && \
        cd "$(mktemp -d)"                                           && \
        KREW="krew-${OS}_${ARCH}"                                   && \
        curl -fsSLO "https://github.com/kubernetes-sigs/krew/releases/latest/download/${KREW}.tar.gz" && \
        tar zxvf "${KREW}.tar.gz"                                   && \
        echo 'export PATH="${KREW_ROOT:-$HOME/.krew}/bin:$PATH"'    >> /root/.bashrc    && \
        ./"${KREW}" install krew                                    && \
        ./"${KREW}" install ctx                                     && \
        ./"${KREW}" install ns                                      && \
        ./"${KREW}" install neat                                    ;\
    fi


ARG INSTALL_KUBEBUILDER
ARG KUBEBUILDER_VER
# Install kubebuilder
RUN if [ "${INSTALL_KUBEBUILDER}" == "true" ]; then \
        export ARCH=$(case $(uname -m) in x86_64) echo -n amd64 ;; aarch64) echo -n arm64 ;; *) echo -n $(uname -m) ;; esac) && \
        export OS=$(uname | awk '{print tolower($0)}') && \
        export KUBEBUILDER_DL_URL=https://github.com/kubernetes-sigs/kubebuilder/releases/download/v${KUBEBUILDER_VER} && \
        curl -LO ${KUBEBUILDER_DL_URL}/kubebuilder_${OS}_${ARCH} && \
        chmod +x kubebuilder_${OS}_${ARCH} && \
        mv kubebuilder_${OS}_${ARCH} /usr/local/bin/kubebuilder; \
    fi


ARG INSTALL_TERRAFORM
ARG TERRAFORM_VER
RUN if [ "${INSTALL_TERRAFORM}" == "true" ]; then \
        export ARCH=$(case $(uname -m) in x86_64) echo -n amd64 ;; aarch64) echo -n arm64 ;; *) echo -n $(uname -m) ;; esac) && \
        export OS=$(uname | awk '{print tolower($0)}') && \
        curl -LO https://releases.hashicorp.com/terraform/${TERRAFORM_VER}/terraform_${TERRAFORM_VER}_${OS}_${ARCH}.zip && \
        unzip terraform_${TERRAFORM_VER}_${OS}_${ARCH}.zip && \
        mv terraform /usr/local/bin/terraform && \
        rm terraform_${TERRAFORM_VER}_${OS}_${ARCH}.zip ;\
    fi

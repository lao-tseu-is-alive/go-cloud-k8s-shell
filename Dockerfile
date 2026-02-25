# Start from the latest golang base image
FROM golang:1.26.0-alpine3.22 AS builder

# Define build arguments for version and build timestamp
ARG APP_REVISION
ARG BUILD
ARG APP_REPOSITORY=https://github.com/lao-tseu-is-alive/go-cloud-k8s-shell

# Add Maintainer Info
LABEL maintainer="cgil"
LABEL org.opencontainers.image.title="go-cloud-k8s-shell"
LABEL org.opencontainers.image.description="This is a go-cloud-k8s-shell container image, a simple Golang microservice with some essential command line tools to make some tests inside a k8s cluster"
LABEL org.opencontainers.image.url="https://ghcr.io/lao-tseu-is-alive/go-cloud-k8s-shell:latest"
LABEL org.opencontainers.image.authors="cgil"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.version="1.0.0"
# Set image version label dynamically
LABEL org.opencontainers.image.source="${APP_REPOSITORY}"

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY "cmd/server" ./server
COPY pkg ./pkg

# Clean the APP_REPOSITORY for ldflags
RUN APP_REPOSITORY_CLEAN=$(echo $APP_REPOSITORY | sed 's|https://||') && \
    CGO_ENABLED=0 GOOS=linux go build -a -ldflags="-w -s -X ${APP_REPOSITORY_CLEAN}/pkg/version.REVISION=${APP_REVISION} -X ${APP_REPOSITORY_CLEAN}/pkg/version.BuildStamp=${BUILD}" -o go-shell-server ./server


######## Start a new stage  #######
FROM ubuntu:26.04
# to comply with security best practices
# Running containers with 'root' user can lead to a container escape situation (the default with Docker...).
# It is a best practice to run containers as non-root users
# https://docs.docker.com/develop/develop-images/dockerfile_best-practices/
# https://docs.docker.com/engine/reference/builder/#user
LABEL author="cgil"
LABEL org.opencontainers.image.authors="cgil"
LABEL description="This is a go-cloud-k8s-shell container image, a simple Golang microservice with some essential command line tools to make some tests inside a k8s cluster "
LABEL org.opencontainers.image.description="This is a go-cloud-k8s-shell container image, a simple Golang microservice with some essential command line tools to make some tests inside a k8s cluster "
LABEL org.opencontainers.image.url="ghcr.io/lao-tseu-is-alive/go-cloud-k8s-shell:latest"
LABEL org.opencontainers.image.source="https://github.com/lao-tseu-is-alive/go-cloud-k8s-shell"

# Pass build arguments to the final stage for labeling
ARG APP_REVISION
ARG BUILD
LABEL org.opencontainers.image.version="${APP_REVISION}"
LABEL org.opencontainers.image.revision="${APP_REVISION}"
LABEL org.opencontainers.image.created="${BUILD}"

RUN apt-get update && apt-get install -y iproute2 file checksec nmap postgresql-client curl jq iputils-ping dnsutils tcpdump iftop netcat-openbsd wget procps && apt-get -y upgrade && apt-get clean && rm -rf /var/lib/apt/lists/*
RUN useradd --create-home --home-dir /home/gouser --shell /bin/bash --user-group --groups users --uid 12221 gouser
RUN groupadd pcap && usermod -a -G pcap gouser && chmod a+x /usr/bin/tcpdump && setcap cap_net_raw,cap_net_admin=eip /usr/bin/tcpdump && setcap cap_net_raw,cap_net_admin=eip  /usr/sbin/iftop
WORKDIR /tmp
RUN curl -LO "https://dl.k8s.io/release/v1.35.1/bin/linux/amd64/kubectl"
RUN install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl
WORKDIR /home/gouser

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/go-shell-server .
COPY scripts/checkK8SApiInsideContainer.sh ./
COPY scripts/getK8SApiFromUrl.sh ./
COPY scripts/getServiceEndPointFromInsideContainer.sh ./
COPY scripts/checkOtherPodConnectivityInsideContainer.sh ./
COPY certificates/isrg-root-x1-cross-signed.pem ./certificates/
RUN chmod a+x ./getK8SApiFromUrl.sh && chmod a+x ./checkK8SApiInsideContainer.sh && chmod a+x ./getServiceEndPointFromInsideContainer.sh &&  chmod a+x ./checkOtherPodConnectivityInsideContainer.sh


# --- Start LS_COLORS configuration ---
RUN dircolors -p > /home/gouser/.dircolors && \
    echo '\n# Configure LS_COLORS for colored `ls` output\n\
if [ -x /usr/bin/dircolors ]; then\n\
    test -r ~/.dircolors && eval "$(dircolors -b ~/.dircolors)" || eval "$(dircolors -b)"\n\
    alias ls="ls --color=auto"\n\
    alias grp="grep --color=auto"\n\
fi' >> /home/gouser/.bashrc && \
    chown gouser:gouser /home/gouser/.dircolors /home/gouser/.bashrc
# --- End LS_COLORS configuration ---

# Switch to non-root user:
USER gouser

RUN echo 'source <(kubectl completion bash)\n\
alias k=kubectl\n\
alias lsa="ls -al -tr"\n\
complete -o default -F __start_kubectl k\n\
APISERVER=https://kubernetes.default.svc \n\
SERVICEACCOUNT=/var/run/secrets/kubernetes.io/serviceaccount \n\
NAMESPACE=$(cat ${SERVICEACCOUNT}/namespace) \n\
TOKEN=$(cat ${SERVICEACCOUNT}/token) \n\
CACERT=${SERVICEACCOUNT}/ca.crt\n\
export CACERT TOKEN NAMESPACE SERVICEACCOUNT APISERVER' >> ~/.bashrc


# Expose port 9999 to the outside world, go-shell-server will use the env PORT as listening port or 9999 as default
EXPOSE 9999

# how to check if container is ok https://docs.docker.com/engine/reference/builder/#healthcheck
HEALTHCHECK --start-period=5s --interval=30s --timeout=3s \
    CMD curl --fail http://localhost:9999/health || exit 1

# Command to run the executable
CMD ["./go-shell-server"]

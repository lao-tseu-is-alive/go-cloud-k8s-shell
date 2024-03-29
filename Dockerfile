# Start from the latest golang base image
FROM golang:1-alpine3.18 AS builder

# Add Maintainer Info
LABEL maintainer="cgil"

#RUN addgroup -S gouser && adduser -S gouser -G gouser
#USER gouser

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY *.go ./

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o go-shell-server .


######## Start a new stage  #######
FROM ubuntu:23.10
# to comply with security best practices
# Running containers with 'root' user can lead to a container escape situation (the default with Docker...).
# It is a best practice to run containers as non-root users
# https://docs.docker.com/develop/develop-images/dockerfile_best-practices/
# https://docs.docker.com/engine/reference/builder/#user

RUN apt-get update && apt-get install -y iproute2 file checksec nmap postgresql-client curl jq iputils-ping dnsutils tcpdump iftop netcat-openbsd wget && apt-get -y upgrade
RUN useradd --create-home --home-dir /home/gouser --shell /bin/bash --user-group --groups users --uid 12221 gouser
RUN groupadd pcap && usermod -a -G pcap gouser && chmod a+x /usr/bin/tcpdump && setcap cap_net_raw,cap_net_admin=eip /usr/bin/tcpdump && setcap cap_net_raw,cap_net_admin=eip  /usr/sbin/iftop
WORKDIR /tmp
#RUN curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
RUN curl -LO "https://dl.k8s.io/release/v1.29.3/bin/linux/amd64/kubectl"
RUN install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl
WORKDIR /home/gouser

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/go-shell-server .
COPY scripts/checkK8SApiInsideContainer.sh ./
COPY scripts/getK8SApiFromUrl.sh ./
COPY scripts/getServiceEndPointFromInsideContainer.sh ./
COPY scripts/checkOtherPodConnectivityInsideContainer.sh ./

RUN chmod a+x ./getK8SApiFromUrl.sh && chmod a+x ./checkK8SApiInsideContainer.sh && chmod a+x ./getServiceEndPointFromInsideContainer.sh &&  chmod a+x ./checkOtherPodConnectivityInsideContainer.sh

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

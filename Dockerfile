# Start from the latest golang base image
FROM golang:1-alpine3.16 as builder

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
FROM ubuntu:22.04
# to comply with security best practices
# Running containers with 'root' user can lead to a container escape situation (the default with Docker...).
# It is a best practice to run containers as non-root users
# https://docs.docker.com/develop/develop-images/dockerfile_best-practices/
# https://docs.docker.com/engine/reference/builder/#user

RUN apt-get update && apt-get install -y iproute2 nmap curl jq && apt-get -y upgrade
RUN useradd --create-home --home-dir /home/gouser --shell /bin/bash --user-group --groups users --uid 1221 gouser

WORKDIR /home/gouser

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/go-shell-server .
COPY scripts/checkK8SApiInsideContainer.sh ./
COPY scripts/getK8SApiFromUrl.sh ./

RUN chmod a+x ./getK8SApiFromUrl.sh && chmod a+x ./checkK8SApiInsideContainer.sh

# Switch to non-root user:
USER gouser



# Expose port 9999 to the outside world, go-shell-server will use the env PORT as listening port or 9999 as default
EXPOSE 9999

# Command to run the executable
CMD ["./go-shell-server"]
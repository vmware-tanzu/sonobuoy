FROM debian:stretch-slim

# Install kubectl
# Note: Latest version may be found on:
# https://aur.archlinux.org/packages/kubectl-bin/
ADD https://storage.googleapis.com/kubernetes-release/release/v1.14.1/bin/linux/amd64/kubectl /usr/local/bin/kubectl

ENV HOME=/config

# Basic check it works.
RUN apt-get update && \
    apt-get -y install net-tools && \
    apt-get -y install curl && \
    chmod +x /usr/local/bin/kubectl && \
    kubectl version --client

COPY ./run.sh ./run.sh 

ENTRYPOINT ["./run.sh"]
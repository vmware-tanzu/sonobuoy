FROM debian:stable-slim
RUN apt-get update && \
    apt-get -y install wget curl git jq && \
    wget -O /bin/sonobuoy.tar.gz https://github.com/vmware-tanzu/sonobuoy/releases/download/v0.56.8/sonobuoy_0.56.8_linux_amd64.tar.gz && \
    tar -C /bin -xzf /bin/sonobuoy.tar.gz && \
    chmod 755 /bin/sonobuoy

WORKDIR /gathere2e

COPY run.sh /gathere2e/run.sh

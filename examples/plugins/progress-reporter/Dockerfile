FROM debian:stretch-slim

# Basic check it works.
RUN apt-get update && \
    apt-get -y install curl

COPY ./run.sh ./run.sh 

ENTRYPOINT ["./run.sh"]
FROM ubuntu

# Install ca-certificates for root certs that we need
RUN apt-get update && \
    apt-get install -y ca-certificates && \
    apt-get clean autoclean && \
    rm -rf /var/lib/{apt,dpkg,cache,log}/

COPY signadot /signadot

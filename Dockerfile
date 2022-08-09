FROM alpine

# Install ca-certificates for root certs that we need
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*

COPY signadot /signadot

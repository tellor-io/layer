FROM golang:1.22-alpine

# Install dependencies
RUN apk add --no-cache bash netcat-openbsd

# Set working directory
WORKDIR /chain

# Copy scripts
COPY start-reporter.sh /chain/start-reporter.sh
COPY entrypoint.sh /chain/entrypoint.sh

# Set proper permissions
RUN chmod +x /chain/start-reporter.sh && \
    chmod +x /chain/entrypoint.sh && \
    mkdir -p /var/log && \
    touch /var/log/validator.log && \
    touch /var/log/reporter.log && \
    chmod 666 /var/log/validator.log && \
    chmod 666 /var/log/reporter.log 
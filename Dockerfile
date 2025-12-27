# Dockerfile for GORM purity survey
# Usage: docker build --build-arg GORM_VERSION=v1.25.0 -t gorm-survey:v1.25.0 .

FROM golang:1.22-alpine

ARG GORM_VERSION=latest

WORKDIR /app

# Install git for go get
RUN apk add --no-cache git

# Copy source files first (excluding go.mod via .dockerignore)
COPY . .

# Use go.mod.template as base (just module declaration)
RUN cp go.mod.template go.mod

# Add replace directive first, then install dependencies
RUN echo "replace gorm.io/gorm => gorm.io/gorm ${GORM_VERSION}" >> go.mod && \
    go get "gorm.io/gorm@${GORM_VERSION}" && \
    go get gorm.io/driver/mysql && \
    go get github.com/DATA-DOG/go-sqlmock && \
    go mod tidy

# Store version for build tag detection
RUN echo "${GORM_VERSION}" > /tmp/gorm_version.txt

# Create entrypoint script that sets build tags based on version
# Generics API (PreloadBuilder, JoinBuilder) was added in v1.30+
RUN echo '#!/bin/sh' > /app/run.sh && \
    echo 'VERSION=$(cat /tmp/gorm_version.txt)' >> /app/run.sh && \
    echo 'MAJOR=$(echo $VERSION | sed "s/v//" | cut -d. -f1)' >> /app/run.sh && \
    echo 'MINOR=$(echo $VERSION | sed "s/v//" | cut -d. -f2)' >> /app/run.sh && \
    echo 'if [ "$MAJOR" -gt 1 ] || ([ "$MAJOR" -eq 1 ] && [ "$MINOR" -ge 30 ]); then' >> /app/run.sh && \
    echo '  TAGS="-tags gorm_v125plus"' >> /app/run.sh && \
    echo 'else' >> /app/run.sh && \
    echo '  TAGS=""' >> /app/run.sh && \
    echo 'fi' >> /app/run.sh && \
    echo 'exec go run $TAGS ./scripts/enumerate/...' >> /app/run.sh && \
    chmod +x /app/run.sh

# Default: run enumeration with appropriate build tags
CMD ["/app/run.sh"]

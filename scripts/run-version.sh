#!/bin/bash
# Run survey for a specific GORM version
# Usage: ./scripts/run-version.sh v1.25.0

set -e

VERSION="${1:?Version required (e.g., v1.25.0)}"
RESULTS_DIR="results"
RESULT_FILE="${RESULTS_DIR}/${VERSION}.json"

# Skip if already processed
if [ -f "$RESULT_FILE" ]; then
    echo "[SKIP] ${VERSION} already processed"
    exit 0
fi

echo "[START] ${VERSION}"

# Build and run container
docker build \
    --build-arg "GORM_VERSION=${VERSION}" \
    --quiet \
    -t "gorm-survey:${VERSION}" \
    . > /dev/null

# Run enumeration (container's entrypoint handles build tags)
docker run --rm "gorm-survey:${VERSION}" > "$RESULT_FILE" 2>&1

echo "[DONE] ${VERSION} -> ${RESULT_FILE}"

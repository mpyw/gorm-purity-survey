#!/bin/bash
# Run survey for a specific GORM version
# Usage: ./scripts/run-version.sh v1.25.0
#
# Includes retry logic for transient Docker/network failures

set -e

VERSION="${1:?Version required (e.g., v1.25.0)}"
METHODS_DIR="methods"
RESULT_FILE="${METHODS_DIR}/${VERSION}.json"
MAX_RETRIES=3

# Skip if already processed
if [ -f "$RESULT_FILE" ]; then
    echo "[SKIP] ${VERSION} already processed"
    exit 0
fi

echo "[START] ${VERSION}"

for attempt in $(seq 1 $MAX_RETRIES); do
    # Build container
    if docker build \
        --build-arg "GORM_VERSION=${VERSION}" \
        --quiet \
        -t "gorm-survey:${VERSION}" \
        . > /dev/null 2>&1; then

        # Run enumeration
        if docker run --rm "gorm-survey:${VERSION}" > "$RESULT_FILE" 2>&1; then
            # Verify JSON is valid
            if jq empty "$RESULT_FILE" 2>/dev/null; then
                echo "[DONE] ${VERSION} -> ${RESULT_FILE}"
                exit 0
            fi
        fi
    fi

    # Failed, retry
    if [ $attempt -lt $MAX_RETRIES ]; then
        echo "[RETRY] ${VERSION} (attempt $((attempt + 1))/${MAX_RETRIES})"
        rm -f "$RESULT_FILE"
        sleep 2
    fi
done

echo "[FAIL] ${VERSION} after ${MAX_RETRIES} attempts"
rm -f "$RESULT_FILE"
exit 1

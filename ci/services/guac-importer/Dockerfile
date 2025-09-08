# Use a lightweight base image, Alpine Linux is a good choice.
FROM alpine:3.19

# --- Build Arguments for Tool Versions ---
# These can be overridden during the docker build command.
# Find latest guacone versions at: https://github.com/guacsec/guac/releases
ARG GUACONE_VERSION="1.0.0"
# Find latest syft versions at: https://github.com/anchore/syft/releases
ARG SYFT_VERSION="1.28.0"

# TARGETARCH is a platform ARG automatically supplied by Docker BuildKit.
# It allows building multi-arch images or for a specific architecture (e.g., amd64, arm64).
ARG TARGETARCH

# --- Environment Setup ---
ENV LANG=C.UTF-8

# --- Install Dependencies ---
# ca-certificates: For HTTPS connections.
# git: For cloning repositories.
# wget: For downloading tool binaries.
# tar: For extracting .tar.gz archives.
RUN apk add --no-cache \
    ca-certificates \
    git \
    wget \
    tar \
    bash # Some scripts might use bash features; sh is default in Alpine

# --- Install guacone ---
# https://github.com/guacsec/guac/releases/download/v1.0.0/guaccollect-linux-amd64
RUN echo "Installing guacone ${GUACONE_VERSION} for architecture ${TARGETARCH}..." && \
    wget "https://github.com/guacsec/guac/releases/download/v${GUACONE_VERSION}/guacone-linux-${TARGETARCH}" \
    -O /usr/local/bin/guacone && \
    chmod +x /usr/local/bin/guacone && \
    guacone --version # Verify installation

# --- Install Syft ---
#https://github.com/anchore/syft/releases/download/v1.28.0/syft_1.28.0_linux_amd64.tar.gz
RUN echo "Installing Syft ${SYFT_VERSION} for architecture ${TARGETARCH}..." && \
    SYFT_PKG_ARCH="${TARGETARCH}" && \
    # Syft uses 'amd64' and 'arm64', Alpine's $TARGETARCH might be 'x86_64'.
    if [ "${TARGETARCH}" = "x86_64" ]; then SYFT_PKG_ARCH="amd64"; fi && \
    if [ "${TARGETARCH}" = "aarch64" ]; then SYFT_PKG_ARCH="arm64"; fi && \
    wget "https://github.com/anchore/syft/releases/download/v${SYFT_VERSION}/syft_${SYFT_VERSION}_linux_${SYFT_PKG_ARCH}.tar.gz" \
    -O /tmp/syft.tar.gz && \
    tar -xzf /tmp/syft.tar.gz -C /usr/local/bin syft && \
    rm /tmp/syft.tar.gz && \
    chmod +x /usr/local/bin/syft && \
    syft version # Verify installation

# --- Application Setup ---
# Set the working directory for the application.
WORKDIR /app

# Copy the ingestion script into the image.
COPY ./ingest-github-data.sh /app/ingest-github-data.sh
RUN chmod +x /app/ingest-github-data.sh

# --- Entrypoint ---
# The script will be executed when the container starts.
ENTRYPOINT ["/app/ingest-github-data.sh"]
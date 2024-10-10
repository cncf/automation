# This exists till https://github.com/actions/runner/pull/3056 is merged
FROM ghcr.io/actions/actions-runner:latest

USER root
RUN apt-get update -y \
    && apt-get install -y --no-install-recommends \
    build-essential \
    npm \
    git \
    curl \
    jq \
    unzip \
    kmod

USER runner

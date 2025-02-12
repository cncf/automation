# This exists till https://github.com/actions/runner/pull/3056 is merged
FROM ghcr.io/actions/actions-runner:2.322.0

USER root
RUN rm /var/lib/dpkg/info/libc-bin.*
RUN apt-get clean
RUN apt-get update -y
RUN apt-get install -y --no-install-recommends libc-bin
RUN apt-get install -y --no-install-recommends build-essential
RUN apt-get install -y --no-install-recommends kmod
RUN apt-get install -y --no-install-recommends zstd

USER runner

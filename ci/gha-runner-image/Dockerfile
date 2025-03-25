# This exists till https://github.com/actions/runner/pull/3056 is merged
FROM ghcr.io/actions/actions-runner:2.323.0

ENV PATH="$PATH:/home/runner/.local/bin"

USER root
RUN apt-get update -y
RUN apt-get install -y --no-install-recommends kmod
RUN apt-get install -y --no-install-recommends zstd
RUN apt-get install -y --no-install-recommends dirmngr
RUN apt-get install -y --no-install-recommends wget
RUN apt-get install -y --no-install-recommends gnupg
RUN apt-get install -y --no-install-recommends python3-pip
RUN apt-get install -y --no-install-recommends locales && \
    locale-gen en_US.UTF-8
RUN apt-get install -y --no-install-recommends make
RUN apt-get install -y --no-install-recommends eatmydata
RUN apt-get install -y --no-install-recommends maven
RUN apt-get install -y --no-install-recommends zip

ENV LANG=en_US.UTF-8 \
    LC_ALL=en_US.UTF-8 \
    LANGUAGE=en_US.UTF-8


USER runner
RUN git config --global --add safe.directory /home/runner/_work
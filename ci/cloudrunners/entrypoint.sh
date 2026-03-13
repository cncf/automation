#!/bin/sh
set -e

case "${CLOUDRUNNER_PROVIDER:-oci}" in
  oci)
    exec /cloudrunner-oci "$@"
    ;;
  kubevirt)
    exec /cloudrunner-kubevirt "$@"
    ;;
  *)
    echo "Unknown CLOUDRUNNER_PROVIDER '${CLOUDRUNNER_PROVIDER}'. Valid values: oci, kubevirt" >&2
    exit 1
    ;;
esac

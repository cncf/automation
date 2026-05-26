#!/bin/sh
set -e

case "${CLOUDRUNNER_PROVIDER:-oci}" in
  oci)
    exec /cloudrunner-oci "$@"
    ;;
  kubevirt)
    exec /cloudrunner-kubevirt "$@"
    ;;
  linode)
    exec /cloudrunner-linode "$@"
    ;;
  *)
    echo "Unknown CLOUDRUNNER_PROVIDER '${CLOUDRUNNER_PROVIDER}'. Valid values: oci, kubevirt, linode" >&2
    exit 1
    ;;
esac

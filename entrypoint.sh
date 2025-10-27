#!/bin/sh

set -e

mkdir -p /tmp/coredock
NAMESERVERS=${COREDOCK_NAMESERVERS:-""}

nameservers="${NAMESERVERS//,/ }"
forward=""

if [ -n "$nameservers" ]; then
  forward="forward . $nameservers"
fi


corefile="
. {
    auto {
        directory /tmp/coredock/
        reload 1s
        ${forward}
    }
}
"
echo "$corefile" > /tmp/coredock/Corefile
coredns -conf /tmp/coredock/Corefile &
./coredock

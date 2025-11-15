#!/bin/sh

set -e

mkdir -p /tmp/coredock
NAMESERVERS=${COREDOCK_NAMESERVERS:-""}

nameservers="${NAMESERVERS//,/ }"
forward=""

forward="fanout . 127.0.0.1:5311"
if [ -n "$nameservers" ]; then
    forward="${forward} ${nameservers} {
    attempt-count 1
    timeout 1s
}"
fi


corefile="
. {
    auto {
        directory /tmp/coredock/
        reload 1s
    }
    loadbalance
}
"

corefileforward="
. {
    ${forward}
}
"

echo "$corefile" > /tmp/coredock/Corefile
echo "$corefileforward" > /tmp/coredock/Corefile.forward
./coredns -dns.port 5311 -p 5311 --conf /tmp/coredock/Corefile &
./coredns --conf /tmp/coredock/Corefile.forward &
./coredock

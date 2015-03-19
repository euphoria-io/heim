#!/bin/sh

IP=$(nslookup $HOSTNAME | tail -n 1 | cut -d ' ' -f 3)
ETCD_CMD="/bin/etcd -name etcd -data-dir /data -addr $IP:4001 $*"
echo "Running: $ETCD_CMD"
$ETCD_CMD

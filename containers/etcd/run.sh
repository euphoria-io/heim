#!/bin/sh

IP=$(ip route get 8.8.8.8 | awk '/8.8.8.8/ {print $NF}')
ETCD_CMD="/bin/etcd -name etcd -data-dir /data -addr $IP:4001 $*"
echo "Running: $ETCD_CMD"
$ETCD_CMD

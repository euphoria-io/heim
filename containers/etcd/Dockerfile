FROM gliderlabs/alpine
MAINTAINER Logan Hanks <logan@euphoria.io>

ADD https://github.com/coreos/etcd/releases/download/v0.4.6/etcd-v0.4.6-linux-amd64.tar.gz etcd-v0.4.6-linux-amd64.tar.gz
RUN tar xzvf etcd-v0.4.6-linux-amd64.tar.gz
RUN mv etcd-v0.4.6-linux-amd64/etcd /bin && mv etcd-v0.4.6-linux-amd64/etcdctl /bin && rm -Rf /etcd-v0.4.6-linux-amd64*

ADD run.sh /bin/run.sh

VOLUME /data
EXPOSE 4001 7001
ENTRYPOINT /bin/run.sh

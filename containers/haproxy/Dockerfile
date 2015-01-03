FROM ubuntu:14.04
MAINTAINER Logan Hanks <logan.hanks@gmail.com>

RUN apt-get update
RUN DEBIAN_FRONTEND=noninteractive apt-get install -y haproxy rsyslog
RUN sed -i 's/^ENABLED=.*/ENABLED=1/' /etc/default/haproxy

ADD haproxy.cfg /etc/haproxy/haproxy.cfg

EXPOSE 80
CMD /etc/init.d/rsyslog start && /etc/init.d/haproxy start && tail -f /var/log/haproxy.log

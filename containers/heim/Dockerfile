FROM ubuntu:14.04
MAINTAINER Logan Hanks <logan.hanks@gmail.com>

RUN apt-get update
RUN DEBIAN_FRONTEND=noninteractive apt-get install -y s3cmd openssh-client
ADD s3cfg /root/.s3cfg

ENV REVISION latest
ENV HZPDIR /hzp

ADD authorized_keys /
ADD sync-and-run.sh /

EXPOSE 22
EXPOSE 80
CMD /sync-and-run.sh

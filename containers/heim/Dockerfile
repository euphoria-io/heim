FROM ubuntu:14.04
MAINTAINER Logan Hanks <logan.hanks@gmail.com>

RUN apt-get update
RUN DEBIAN_FRONTEND=noninteractive apt-get install -y s3cmd
ADD s3cfg /root/.s3cfg

ENV REVISION latest

ADD sync-and-run.sh /

EXPOSE 80
CMD /sync-and-run.sh

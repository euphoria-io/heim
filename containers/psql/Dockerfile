FROM postgres:9.4
MAINTAINER Logan Hanks <logan.hanks@gmail.com>

ENV POSTGRES_PASSWORD heimlich

RUN mkdir -p /docker-entrypoint-initdb.d
ADD create-heim.sh /docker-entrypoint-initdb.d/create-heim.sh

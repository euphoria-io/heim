FROM postgres:9.4
MAINTAINER Logan Hanks <logan@euphoria.io>

ENV POSTGRES_PASSWORD heimlich

RUN mkdir -p /docker-entrypoint-initdb.d
ADD create-heim.sql /docker-entrypoint-initdb.d/create-heim.sql

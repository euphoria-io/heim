#!/bin/sh

gosu postgres postgres --single << EOM
  CREATE DATABASE heim;
EOM

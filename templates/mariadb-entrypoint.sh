#!/bin/bash

/usr/local/bin/docker-gen mariadb.tmpl /tmp/sql.sh
while true;
do
        ls -l /tmp/sql.sh
        source /tmp/sql.sh
        /usr/local/bin/docker-gen --watch mariadb.tmpl /tmp/sql.sh
done

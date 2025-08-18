FROM jwilder/docker-gen AS docker-gen
FROM mariadb:latest
COPY --from=docker-gen /usr/local/bin/docker-gen /usr/local/bin/docker-gen
COPY mariadb-entrypoint.sh /
COPY mariadb.tmpl /
ENTRYPOINT ["/mariadb-entrypoint.sh"]

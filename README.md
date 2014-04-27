docker-gen
=====

`docker-gen` is a file generator that renders templates using docker container meta-data.

docker-gen can be used to generate various kinds of files for:

 * **Centralized logging** - fluentd, logstash or other centralized logging tools that tail the containers JSON log file or files within the container.
 * **Log Roatation** - logrotate files to rotate container JSON log files
 * **Reverse Proxy Configs** - nginx, haproxy, etc. reverse proxy configs to route requests from the host to containers
 * **Service Discovery** - Scripts (python, bash, etc..) to register containers within etcd, hipache, etc..

===

### Installation

```
go get github.com/jwilder/docker-gen
```

### Usage
```
docker-gen
Usage: docker-gen [-config file] [-watch=false] [-notify="restart xyz"] [-interval=0] <template> [<dest>]
```

*Options:*
* `-watch` - runs continuously and monitors docker container events.  When containers are started
or stopped, the template is regenerated.
* `-notify` - runs a command after the template is generated.  Useful for restarting nginx, reloading
haproxy, etc..
* `-config file` - Use the specified config file instead of command-line options.  Multiple templates can be defined and they will be executed in the order that they appear in the config file.
* `-interval <secs>` - Run the notify command on a fixed interval.  Useful for service registration use cases.

If no `<dest>` file is specified, the output is sent to stdout.  Mainly useful for debugging.


### Examples

#### NGINX Reverse Proxy Config

```
docker-gen -only-exposed -watch -notify "/etc/init.d/nginx reload" templates/nginx.tmpl /etc/nginx/sites-enabled/default
```

[Automated Nginx Reverse Proxy for Docker](http://jasonwilder.com/blog/2014/03/25/automated-nginx-reverse-proxy-for-docker/)

#### Fluentd Log Management

```
docker-gen -watch -notify "restart fluentd" templates/fluentd.tmpl /etc/fluent/fluent.conf
```

[Docker Log Management With Fluentd](http://jasonwilder.com/blog/2014/03/17/docker-log-management-using-fluentd/)

#### Register Containers in Etcd

```
docker-gen -notify "/bin/bash /tmp/etcd.sh" -interval 10 templates/etcd.tmpl /tmp/etcd.sh
```


### TODO

 * Add a way to filter out containers in templates

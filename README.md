docker-gen
=====

`docker-gen` is a file generator that renders templates using docker container meta-data.

docker-gen can be used to generate various kinds of files for:

 * **Centralized logging** - fluentd, logstash or other centralized logging tools that tail the containers JSON log file or files within the container.
 * **Log Rotation** - logrotate files to rotate container JSON log files
 * **Reverse Proxy Configs** - nginx, haproxy, etc. reverse proxy configs to route requests from the host to containers
 * **Service Discovery** - Scripts (python, bash, etc..) to register containers within etcd, hipache, etc..

===

### Installation

#### Host Install

Linux binaries for release [0.3.1](https://github.com/jwilder/docker-gen/releases)

* [amd64](https://github.com/jwilder/docker-gen/releases/download/0.3.1/docker-gen-linux-amd64-0.3.1.tar.gz)
* [i386](https://github.com/jwilder/docker-gen/releases/download/0.3.1/docker-gen-linux-i386-0.3.1.tar.gz)

Download the version you need, untar, and install to your PATH.

```
$ wget https://github.com/jwilder/docker-gen/releases/download/0.3.1/docker-gen-linux-amd64-0.3.1.tar.gz
$ tar xvzf docker-gen-linux-amd64-0.3.1.tar.gz
$ ./docker-gen
```

#### Container Install

See [jwilder/nginx-proxy](https://index.docker.io/u/jwilder/nginx-proxy/) trusted build as an example of running docker-gen within a container.


### Usage
```
$ docker-gen
Usage: docker-gen [options] <template> [<dest>]
```

[-config file] [-watch=false] [-notify="restart xyz"] [-interval=0] [-endpoint tcp|unix://..]

*Options:*
```
  -config="": Use the specified config file instead of command-line options.  Multiple templates can be defined and 
              they will be executed in the order that they appear in the config file.
  -endpoint="": docker api endpoint [tcp|unix://..]. This can also be set w/ a `DOCKER_HOST` environment.
  -interval=0:run notify command interval (s). Useful for service registration use cases.
  -notify="": run command after template is regenerated ["restart xyz"]. Useful for restarting nginx,
              reloading haproxy, etc..
  -only-exposed=false: only include containers with exposed ports
  -only-published=false: only include containers with published ports (implies -only-exposed)
  -version=false: show version
  -watch=false: run continuously and monitors docker container events.  When containers are started
                or stopped, the template is regenerated.
```

If no `<dest>` file is specified, the output is sent to stdout.  Mainly useful for debugging.


### Examples

#### NGINX Reverse Proxy Config

##### Containerized

[jwilder/nginx-proxy](https://index.docker.io/u/jwilder/nginx-proxy/) trusted build.

Start nginx-proxy:

```
$ docker run -d -p 80:80 -v /var/run/docker.sock:/tmp/docker.sock -t jwilder/nginx-proxy
```

Then start containers with a VIRTUAL_HOST env variable:
```
$ docker run -e VIRTUAL_HOST=foo.bar.com -t ...
```

##### Host Install

```
$ docker-gen -only-exposed -watch -notify "/etc/init.d/nginx reload" templates/nginx.tmpl /etc/nginx/sites-enabled/default
```

[Automated Nginx Reverse Proxy for Docker](http://jasonwilder.com/blog/2014/03/25/automated-nginx-reverse-proxy-for-docker/)

#### Fluentd Log Management

```
$ docker-gen -watch -notify "restart fluentd" templates/fluentd.tmpl /etc/fluent/fluent.conf
```

[Docker Log Management With Fluentd](http://jasonwilder.com/blog/2014/03/17/docker-log-management-using-fluentd/)

#### Register Containers in Etcd

```
$ docker-gen -notify "/bin/bash /tmp/etcd.sh" -interval 10 templates/etcd.tmpl /tmp/etcd.sh
```


### Development

This project uses [godep](https://github.com/tools/godep) for managing 3rd party dependencies.  You'll need to install godep into your workspace before hacking on docker-gen.

```
$ git clone <your fork>
$ godep restore
$ make
```

### TODO

 * Add event status for handling start and stop events differently
 * Add a way to filter out containers in templates

docker-gen
=====

![latest v0.3.6](https://img.shields.io/badge/latest-v0.3.6-green.svg?style=flat)
[![Build Status](https://travis-ci.org/jwilder/docker-gen.svg?branch=master)](https://travis-ci.org/jwilder/docker-gen)
![License MIT](https://img.shields.io/badge/license-MIT-blue.svg?style=flat)

`docker-gen` is a file generator that renders templates using docker container meta-data.

It can be used to generate various kinds of files for:

 * **Centralized logging** - [fluentd](https://github.com/jwilder/docker-gen/blob/master/templates/fluentd.conf.tmpl), logstash or other centralized logging tools that tail the containers JSON log file or files within the container.
 * **Log Rotation** - [logrotate](https://github.com/jwilder/docker-gen/blob/master/templates/logrotate.tmpl) files to rotate container JSON log files
 * **Reverse Proxy Configs** - [nginx](https://github.com/jwilder/docker-gen/blob/master/templates/nginx.tmpl), [haproxy](https://github.com/jwilder/docker-discover), etc. reverse proxy configs to route requests from the host to containers
 * **Service Discovery** - Scripts (python, bash, etc..) to register containers within [etcd](https://github.com/jwilder/docker-register), hipache, etc..

===

### Installation

There are three common ways to run docker-gen:
* on the host
* bundled in a container with another application
* separate standalone containers

#### Host Install

Linux binaries for release [0.3.6](https://github.com/jwilder/docker-gen/releases)

* [amd64](https://github.com/jwilder/docker-gen/releases/download/0.3.6/docker-gen-linux-amd64-0.3.6.tar.gz)
* [i386](https://github.com/jwilder/docker-gen/releases/download/0.3.6/docker-gen-linux-i386-0.3.6.tar.gz)

Download the version you need, untar, and install to your PATH.

```
$ wget https://github.com/jwilder/docker-gen/releases/download/0.3.6/docker-gen-linux-amd64-0.3.6.tar.gz
$ tar xvzf docker-gen-linux-amd64-0.3.6.tar.gz
$ ./docker-gen
```

#### Bundled Container Install

Docker-gen can be bundled inside of a container along-side and applications.

[jwilder/nginx-proxy](https://index.docker.io/u/jwilder/nginx-proxy/) trusted build is an example of
running docker-gen within a container along-side nginx.
[jwilder/docker-register](https://github.com/jwilder/docker-register) is an example of running
docker-gen within a container to do service registration with etcd.

#### Separate Container Install

It can also be run as two separate containers using the [jwilder/docker-gen](https://index.docker.io/u/jwilder/docker-gen/)
image virtually any other image.

This is how you could run the official [nginx](https://registry.hub.docker.com/_/nginx/) image and
have dockgen-gen generate a reverse proxy config in the same way that `nginx-proxy` works.  You may want to do
this to prevent having the docker socket bound to an publicly exposed container service.

Start nginx with a shared volume:

```
$ docker run -d -p 80:80 --name nginx -v /tmp/nginx:/etc/nginx/conf.d -t nginx
```

Fetch the template and start the docker-gen container with the shared volume:
```
$ mkdir -p /tmp/templates && cd /tmp/templates
$ curl -o nginx.tmpl https://raw.githubusercontent.com/jwilder/docker-gen/master/templates/nginx.tmpl
$ docker run -d --name nginx-gen --volumes-from nginx \
   -v /var/run/docker.sock:/tmp/docker.sock \
   -v /tmp/templates:/etc/docker-gen/templates \
   -t jwilder/docker-gen:0.3.4 -notify-sighup nginx -watch --only-published /etc/docker-gen/templates/nginx.tmpl /etc/nginx/conf.d/default.conf
```

===

### Usage
```
$ docker-gen
Usage: docker-gen [-config file] [-watch=false] [-notify="restart xyz"] [-notify-sighup="nginx-proxy"] [-interval=0] [-endpoint tcp|unix://..] [-tlsverify] [-tlscert file] [-tlskey file] [-tlscacert file] <template> [<dest>]
```

*Options:*

```
  -config="": Use the specified config file instead of command-line options.  Multiple templates can be defined and
              they will be executed in the order that they appear in the config file.
  -endpoint="": docker api endpoint [tcp|unix://..]. This can also be set w/ a `DOCKER_HOST` environment.
  -interval=0:run notify command interval (s). Useful for service registration use cases.
  -notify="": run command after template is regenerated ["restart xyz"]. Useful for restarting nginx,
              reloading haproxy, etc..
  -notify-sighup="": send HUP signal to container.  Equivalent to `docker kill -s HUP container-ID`
  -only-exposed=false: only include containers with exposed ports
  -only-published=false: only include containers with published ports (implies -only-exposed)
  -tlscacert="": path to TLS CA certificate file
  -tlscert="": path to TLS client certificate file
  -tlskey="": path to TLS client key file
  -tlsverify=false: verify docker daemon's TLS certicate
  -version=false: show version
  -watch=false: run continuously and monitors docker container events.  When containers are started
                or stopped, the template is regenerated.
```

If no `<dest>` file is specified, the output is sent to stdout.  Mainly useful for debugging.

===

### Templating

The templates used by docker-gen are written using the Go [text/template](http://golang.org/pkg/text/template/) language. In addition to the [built-in functions](http://golang.org/pkg/text/template/#hdr-Functions) supplied by Go, docker-gen provides a number of additional functions to make it simpler (or possible) to generate your desired output.

Within those templates, the object emitted by docker-gen will have [this structure](https://github.com/jwilder/docker-gen/wiki/Docker-Gen-Emit-Structure).

#### Functions

* *`closest $array $value`: Returns the longest matching substring in `$array` that matches `$value`
* *`coalesce ...`*: Returns the first non-nil argument.
* *`contains $map $key`*: Returns `true` if `$map` contains `$key`. Takes maps from `string` to `string`.
* *`dict $key $value ...`*: Creates a map from a list of pairs. Each `$key` value must be a `string`, but the `$value` can be any type (or `nil`). Useful for passing more than one value as a pipeline context to subtemplates.
* *`dir $path`: Returns an array of filenames in the specified `$path`.
* *`exists $path`*: Returns `true` if `$path` refers to an existing file or directory. Takes a string.
* *`first $array`*: Returns the first value of an array or nil if the arry is nil or empty.
* *`groupBy $containers $fieldPath`*: Groups an array of `RuntimeContainer` instances based on the values of a field path expression `$fieldPath`. A field path expression is a dot-delimited list of map keys or struct member names specifying the path from container to a nested value, which must be a string. Returns a map from the value of the field path expression to an array of containers having that value. Containers that do not have a value for the field path in question are omitted.
* *`groupByKeys $containers $fieldPath`*: Returns the same as `groupBy` but only returns the keys of the map.
* *`groupByMulti $containers $fieldPath $sep`*: Like `groupBy`, but the string value specified by `$fieldPath` is first split by `$sep` into a list of strings. A container whose `$fieldPath` value contains a list of strings will show up in the map output under each of those strings.
* *`hasPrefix $prefix $string`*: Returns whether `$prefix` is a prefix of `$string`.
* *`hasSuffix $suffix $string`*: Returns whether `$suffix` is a suffix of `$string`.
* *`intersect $slice1 $slice2`*: Returns the strings that exist in both string slices.
* *`json $value`*: Returns the JSON representation of `$value` as a `string`.
* *`keys $map`*: Returns the keys from `$map`. If `$map` is `nil`, a `nil` is returned. If `$map` is not a `map`, an error will be thrown.
* *`last $array`*: Returns the last value of an array.
* *`replace $string $old $new $count`*: Replaces up to `$count` occurences of `$old` with `$new` in `$string`. Alias for [`strings.Replace`](http://golang.org/pkg/strings/#Replace)
* *`sha1 $string`*: Returns the hexadecimal representation of the SHA1 hash of `$string`.
* *`split $string $sep`*: Splits `$string` into a slice of substrings delimited by `$sep`. Alias for [`strings.Split`](http://golang.org/pkg/strings/#Split)
* *`trimPrefix $prefix $string`*: If `$prefix` is a prefix of `$string`, return `$string` with `$prefix` trimmed from the beginning. Otherwise, return `$string` unchanged.
* *`trimSuffix $suffix $string`*: If `$suffix` is a suffix of `$string`, return `$string` with `$suffix` trimmed from the end. Otherwise, return `$string` unchanged.
* *`where $containers $fieldPath $value`*: Filters an array of `RuntimeContainer` instances based on the values of a field path expression `$fieldPath`. A field path expression is a dot-delimited list of map keys or struct member names specifying the path from container to a nested value, which must be a string. Returns an array of containers having that value.
* *`whereAny $containers $fieldPath $sep $values`*: Like `where`, but the string value specified by `$fieldPath` is first split by `$sep` into a list of strings. The comparison value is a string slice with possible matches. Returns containers which OR intersect these values.
* *`whereAll $containers $fieldPath $sep $values`*: Like `whereAny`, except all `$values` must exist in the `$fieldPath`.

===

### Examples

* [Automated Nginx Reverse Proxy for Docker](http://jasonwilder.com/blog/2014/03/25/automated-nginx-reverse-proxy-for-docker/)
* [Docker Log Management With Fluentd](http://jasonwilder.com/blog/2014/03/17/docker-log-management-using-fluentd/)
* [Docker Service Discovery Using Etcd and Haproxy](http://jasonwilder.com/blog/2014/07/15/docker-service-discovery/)

#### NGINX Reverse Proxy Config

[jwilder/nginx-proxy](https://index.docker.io/u/jwilder/nginx-proxy/) trusted build.

Start nginx-proxy:

```
$ docker run -d -p 80:80 -v /var/run/docker.sock:/tmp/docker.sock -t jwilder/nginx-proxy
```

Then start containers with a VIRTUAL_HOST env variable:

```
$ docker run -e VIRTUAL_HOST=foo.bar.com -t ...
```

If you wanted to run docker-gen directly on the host, you could do it with:

```
$ docker-gen -only-published -watch -notify "/etc/init.d/nginx reload" templates/nginx.tmpl /etc/nginx/sites-enabled/default
```

#### Fluentd Log Management

This template generate a fluentd.conf file used by fluentd.  It would then ships log files off
the host.

```
$ docker-gen -watch -notify "restart fluentd" templates/fluentd.tmpl /etc/fluent/fluent.conf
```

#### Service Discovery in Etcd


This template is an example of generating a script that is then executed.  This tempalte generates
a python script that is then executed which register containers in Etcd using it's HTTP API.

```
$ docker-gen -notify "/bin/bash /tmp/etcd.sh" -interval 10 templates/etcd.tmpl /tmp/etcd.sh
```


### Development

This project uses [glock](https://github.com/robfig/glock) for managing 3rd party dependencies.
You'll need to install glock into your workspace before hacking on docker-gen.

```
$ git clone <your fork>
$ glock sync github.com/jwilder/docker-gen
$ make
```

### TODO

 * Add event status for handling start and stop events differently

### License

MIT
 

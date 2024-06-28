# docker-gen

[![Tests](https://github.com/nginx-proxy/docker-gen/actions/workflows/tests.yml/badge.svg)](https://github.com/nginx-proxy/docker-gen/actions/workflows/tests.yml)
[![GitHub release](https://img.shields.io/github/v/release/nginx-proxy/docker-gen)](https://github.com/nginx-proxy/docker-gen/releases)
[![Docker Image Size](https://img.shields.io/docker/image-size/nginxproxy/docker-gen?sort=semver)](https://hub.docker.com/r/nginxproxy/docker-gen "Click to view the image on Docker Hub")
[![Docker stars](https://img.shields.io/docker/stars/nginxproxy/docker-gen.svg)](https://hub.docker.com/r/nginxproxy/docker-gen "DockerHub")
[![Docker pulls](https://img.shields.io/docker/pulls/nginxproxy/docker-gen.svg)](https://hub.docker.com/r/nginxproxy/docker-gen "DockerHub")

`docker-gen` is a file generator that renders templates using docker container meta-data.

It can be used to generate various kinds of files for:

- **Centralized logging** - [fluentd](https://github.com/nginx-proxy/docker-gen/blob/main/templates/fluentd.conf.tmpl), logstash or other centralized logging tools that tail the containers JSON log file or files within the container.
- **Log Rotation** - [logrotate](https://github.com/nginx-proxy/docker-gen/blob/main/templates/logrotate.tmpl) files to rotate container JSON log files
- **Reverse Proxy Configs** - [nginx](https://github.com/nginx-proxy/docker-gen/blob/main/templates/nginx.tmpl), [haproxy](https://github.com/jwilder/docker-discover), etc. reverse proxy configs to route requests from the host to containers
- **Service Discovery** - Scripts (python, bash, etc..) to register containers within [etcd](https://github.com/jwilder/docker-register), hipache, etc..

---

### Installation

There are three common ways to run docker-gen:

- on the host
- bundled in a container with another application
- separate standalone containers

#### Host Install

Download the version you need, untar, and install to your PATH.

```console
wget https://github.com/nginx-proxy/docker-gen/releases/download/0.12.0/docker-gen-linux-amd64-0.12.0.tar.gz
tar xvzf docker-gen-linux-amd64-0.12.0.tar.gz
./docker-gen
```

#### Bundled Container Install

Docker-gen can be bundled inside of a container along-side applications.

[nginx-proxy/nginx-proxy](https://hub.docker.com/r/nginxproxy/nginx-proxy) trusted build is an example of
running docker-gen within a container along-side nginx.
[jwilder/docker-register](https://github.com/jwilder/docker-register) is an example of running
docker-gen within a container to do service registration with etcd.

#### Separate Container Install

It can also be run as two separate containers using the [nginx-proxy/docker-gen](https://hub.docker.com/r/nginxproxy/docker-gen)
image, together with virtually any other image.

This is how you could run the official [nginx](https://registry.hub.docker.com/_/nginx/) image and
have docker-gen generate a reverse proxy config in the same way that `nginx-proxy` works. You may want to do
this to prevent having the docker socket bound to a publicly exposed container service.

Start nginx with a shared volume:

```console
docker run -d -p 80:80 --name nginx -v /tmp/nginx:/etc/nginx/conf.d -t nginx
```

Fetch the template and start the docker-gen container with the shared volume:

```console
mkdir -p /tmp/templates && cd /tmp/templates
curl -o nginx.tmpl https://raw.githubusercontent.com/nginx-proxy/docker-gen/main/templates/nginx.tmpl
docker run -d --name nginx-gen --volumes-from nginx \
   -v /var/run/docker.sock:/tmp/docker.sock:rw \
   -v /tmp/templates:/etc/docker-gen/templates \
   -t nginxproxy/docker-gen -notify-sighup nginx -watch -only-exposed /etc/docker-gen/templates/nginx.tmpl /etc/nginx/conf.d/default.conf
```

Start a container, taking note of any Environment variables a container expects. See the top of a template for details.

```console
docker run --env VIRTUAL_HOST='example.com' --env VIRTUAL_PORT=80 ...
```

---

### Usage

```
$ docker-gen
Usage: docker-gen [options] template [dest]

Generate files from docker container meta-data

Options:
  -config path
      config files with template directives.
      Config files will be merged if this option is specified multiple times. (default [])
  -container-filter key=value
      container filter for inclusion by docker-gen.
      You can pass this option multiple times to combine filters with AND.
      https://docs.docker.com/engine/reference/commandline/ps/#filter
  -endpoint string
      docker api endpoint (tcp|unix://..). Default unix:///var/run/docker.sock
  -include-stopped
      include stopped containers.
      Bypassed by when providing a container status filter (-container-filter status=foo).
  -interval int
      notify command interval (secs)
  -keep-blank-lines
      keep blank lines in the output file
  -notify restart xyz
      run command after template is regenerated (e.g restart xyz)
  -notify-container container-ID
      send -notify-signal signal (defaults to 1 / HUP) to container.
      You can pass this option multiple times to notify multiple containers.
  -notify-filter key=value
      container filter for notification (e.g -notify-filter name=foo).
      You can pass this option multiple times to combine filters with AND.
      https://docs.docker.com/engine/reference/commandline/ps/#filter
  -notify-output
      log the output(stdout/stderr) of notify command
  -notify-sighup container-ID
      send HUP signal to container.
      Equivalent to 'docker kill -s HUP container-ID', or `-notify-container container-ID -notify-signal 1`.
      You can pass this option multiple times to send HUP to multiple containers.
  -notify-signal signal
      signal to send to the -notify-container and -notify-filter. -1 to call docker restart. Defaults to 1 aka. HUP.
      All available signals available on the dockerclient
      https://github.com/fsouza/go-dockerclient/blob/main/signal.go
  -only-exposed
      only include containers with exposed ports.
      Bypassed by when using the exposed filter with (-container-filter exposed=foo).
  -only-published
      only include containers with published ports (implies -only-exposed).
      Bypassed by when providing a container published filter (-container-filter published=foo).
  -tlscacert string
      path to TLS CA certificate file (default "~/.docker/ca.pem")
  -tlscert string
      path to TLS client certificate file (default "~/.docker/cert.pem")
  -tlskey string
      path to TLS client key file (default "~/.docker/key.pem")
  -tlsverify
      verify docker daemon's TLS certicate
  -version
      show version
  -wait string
      minimum and maximum durations to wait (e.g. "500ms:2s") before triggering generate
  -watch
      watch for container changes

Arguments:
  template - path to a template to generate
  dest - path to write the template to. If not specfied, STDOUT is used

Environment Variables:
  DOCKER_HOST - default value for -endpoint
  DOCKER_CERT_PATH - directory path containing key.pem, cert.pem and ca.pem
  DOCKER_TLS_VERIFY - enable client TLS verification
```

If no `<dest>` file is specified, the output is sent to stdout. Mainly useful for debugging.

### Configuration file

Using the -config flag from above you can tell docker-gen to use the specified config file instead of command-line options. Multiple templates can be defined and they will be executed in the order that they appear in the config file.

An example configuration file, **docker-gen.cfg** can be found in the examples folder.

#### Configuration File Syntax

```ini
[[config]]
# Starts a configuration section

dest = "path/to/a/file"
# path to write the template. If not specfied, STDOUT is used

notifycmd = "/etc/init.d/foo reload"
# run command after template is regenerated (e.g restart xyz)

onlyexposed = true
# only include containers with exposed ports

template = "/path/to/a/template/file.tmpl"
# path to a template to generate

watch = true
# watch for container changes

wait = "500ms:2s"
# debounce changes with a min:max duration. Only applicable if watch = true


[config.NotifyContainers]
# Starts a notify container section

containername = 1
# container name followed by the signal to send

container_id = 1
# or the container id can be used followed by the signal to send
```

Putting it all together here is an example configuration file.

```ini
[[config]]
template = "/etc/nginx/nginx.conf.tmpl"
dest = "/etc/nginx/sites-available/default"
onlyexposed = true
notifycmd = "/etc/init.d/nginx reload"

[[config]]
template = "/etc/logrotate.conf.tmpl"
dest = "/etc/logrotate.d/docker"
watch = true

[[config]]
template = "/etc/docker-gen/templates/nginx.tmpl"
dest = "/etc/nginx/conf.d/default.conf"
watch = true
wait = "500ms:2s"

[config.NotifyContainers]
nginx = 1  # 1 is a signal number to be sent; here SIGHUP
e75a60548dc9 = 1  # a key can be either container name (nginx) or ID
```

---

### Templating

The templates used by docker-gen are written using the Go [text/template](http://golang.org/pkg/text/template/) language. In addition to the [built-in functions](http://golang.org/pkg/text/template/#hdr-Functions) supplied by Go, docker-gen uses [sprig](https://masterminds.github.io/sprig/) and some additional functions to make it simpler (or possible) to generate your desired output. Some templates rely on environment variables within the container to make decisions on what to generate from the template.

Several templates may be parsed at once by using a semicolon (`;`) to delimit the `template` value. This can be used as a proxy for Golang's nested template functionality. In all cases, the main rendered template should go first.

```
[[config]]
template = "/etc/docker-gen/templates/nginx.tmpl;/etc/docker-gen/templates/header.tmpl"
dest = "/etc/nginx/conf.d/default.conf"
watch = true
wait = "500ms:2s"
```

#### Emit Structure

Within the templates, the object emitted by docker-gen will be a structure consisting of following Go structs:

```go
type RuntimeContainer struct {
    ID           string
    Created      time.Time
    Addresses    []Address
    Networks     []Network
    Gateway      string
    Name         string
    Hostname     string
    Image        DockerImage
    Env          map[string]string
    Volumes      map[string]Volume
    Node         SwarmNode
    Labels       map[string]string
    IP           string
    IP6LinkLocal string
    IP6Global    string
    Mounts       []Mount
    State        State
}

type Address struct {
    IP           string
    IP6LinkLocal string
    IP6Global    string
    Port         string
    HostPort     string
    Proto        string
    HostIP       string
}

type Network struct {
    IP                  string
    Name                string
    Gateway             string
    EndpointID          string
    IPv6Gateway         string
    GlobalIPv6Address   string
    MacAddress          string
    GlobalIPv6PrefixLen int
    IPPrefixLen         int
    Internal            bool
}

type DockerImage struct {
    Registry   string
    Repository string
    Tag        string
}

type Mount struct {
  Name        string
  Source      string
  Destination string
  Driver      string
  Mode        string
  RW          bool
}

type Volume struct {
    Path      string
    HostPath  string
    ReadWrite bool
}

type SwarmNode struct {
    ID      string
    Name    string
    Address Address
}

type State struct {
	Running bool
	Health  Health
}

type Health struct {
	Status string
}

// Accessible from the root in templates as .Docker
type Docker struct {
    Name                 string
    NumContainers        int
    NumImages            int
    Version              string
    ApiVersion           string
    GoVersion            string
    OperatingSystem      string
    Architecture         string
    CurrentContainerID   string
}

// Host environment variables accessible from root in templates as .Env
```

For example, this is a JSON version of an emitted RuntimeContainer struct:

```json
{
  "ID": "71e9768075836eb38557adcfc71a207386a0c597dbeda240cf905df79b18cebf",
  "Addresses": [
    {
      "IP": "172.17.0.4",
      "Port": "22",
      "Proto": "tcp",
      "HostIP": "192.168.10.24",
      "HostPort": "2222"
    }
  ],
  "Gateway": "172.17.42.1",
  "Node": {
    "ID": "I2VY:P7PF:TZD5:PGWB:QTI7:QDSP:C5UD:DYKR:XKKK:TRG2:M2BL:DFUN",
    "Name": "docker-test",
    "Address": {
      "IP": "192.168.10.24"
    }
  },
  "Labels": {
    "operatingsystem": "Ubuntu 14.04.2 LTS",
    "storagedriver": "devicemapper",
    "anything_foo": "something_bar"
  },
  "IP": "172.17.0.4",
  "Name": "docker_register",
  "Hostname": "71e976807583",
  "Image": {
    "Registry": "jwilder",
    "Repository": "docker-register"
  },
  "Env": {
    "ETCD_HOST": "172.17.42.1:4001",
    "PATH": "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
    "DOCKER_HOST": "unix:///var/run/docker.sock",
    "HOST_IP": "172.17.42.1"
  },
  "Volumes": {
    "/mnt": {
      "Path": "/mnt",
      "HostPath": "/Users/joebob/tmp",
      "ReadWrite": true
    }
  }
}
```

#### Functions

- [Functions from Go](https://pkg.go.dev/text/template#hdr-Functions)
- [Functions from Sprig v3](https://masterminds.github.io/sprig/), except for those that have the same name as one of the following functions.
- _`closest $array $value`_: Returns the longest matching substring in `$array` that matches `$value`
- _`coalesce ...`_: Returns the first non-nil argument.
- _`comment $delimiter $string`_: Returns `$string` with each line prefixed by `$delimiter` (helpful for debugging combined with Sprig `toPrettyJson`: `{{ toPrettyJson $ | comment "#" }}`).
- _`contains $map $key`_: Returns `true` if `$map` contains `$key`. Takes maps from `string` to any type.
- _`dir $path`_: Returns an array of filenames in the specified `$path`.
- _`exists $path`_: Returns `true` if `$path` refers to an existing file or directory. Takes a string.
- _`eval $templateName [$data]`_: Evaluates the named template like Go's built-in `template` action, but instead of writing out the result it returns the result as a string so that it can be post-processed. The `$data` argument may be omitted, which is equivalent to passing `nil`.
- _`groupBy $containers $fieldPath`_: Groups an array of `RuntimeContainer` instances based on the values of a field path expression `$fieldPath`. A field path expression is a dot-delimited list of map keys or struct member names specifying the path from container to a nested value, which must be a string. Returns a map from the value of the field path expression to an array of containers having that value. Containers that do not have a value for the field path in question are omitted.
- _`groupByWithDefault $containers $fieldPath $defaultValue`_: Returns the same as `groupBy`, but containers that do not have a value for the field path are instead included in the map under the `$defaultValue` key.
- _`groupByKeys $containers $fieldPath`_: Returns the same as `groupBy` but only returns the keys of the map.
- _`groupByMulti $containers $fieldPath $sep`_: Like `groupBy`, but the string value specified by `$fieldPath` is first split by `$sep` into a list of strings. A container whose `$fieldPath` value contains a list of strings will show up in the map output under each of those strings.
- _`groupByLabel $containers $label`_: Returns the same as `groupBy` but grouping by the given label's value. Containers that do not have the `$label` set are omitted.
- _`groupByLabelWithDefault $containers $label $defaultValue`_: Returns the same as `groupBy` but grouping by the given label's value. Containers that do not have the `$label` set are included in the map under the `$defaultValue` key.
- _`include $file`_: Returns content of `$file`, and empty string if file reading error.
- _`intersect $slice1 $slice2`_: Returns the strings that exist in both string slices.
- _`json $value`_: Returns the JSON representation of `$value` as a `string`.
- _`fromYaml $string` / `mustFromYaml $string`_: Similar to [Sprig's `fromJson` / `mustFromJson`](https://github.com/Masterminds/sprig/blob/master/docs/defaults.md#fromjson-mustfromjson), but for YAML.
- _`toYaml $dict` / `mustToYaml $dict`_: Similar to [Sprig's `toJson` / `mustToJson`](https://github.com/Masterminds/sprig/blob/master/docs/defaults.md#tojson-musttojson), but for YAML.
- _`keys $map`_: Returns the keys from `$map`. If `$map` is `nil`, a `nil` is returned. If `$map` is not a `map`, an error will be thrown.
- _`parseBool $string`_: parseBool returns the boolean value represented by the string. It accepts 1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False. Any other value returns an error. Alias for [`strconv.ParseBool`](http://golang.org/pkg/strconv/#ParseBool)
- _`replace $string $old $new $count`_: Replaces up to `$count` occurences of `$old` with `$new` in `$string`. Alias for [`strings.Replace`](http://golang.org/pkg/strings/#Replace)
- _`sha1 $string`_: Returns the hexadecimal representation of the SHA1 hash of `$string`.
- _`split $string $sep`_: Splits `$string` into a slice of substrings delimited by `$sep`. Alias for [`strings.Split`](http://golang.org/pkg/strings/#Split)
- _`splitN $string $sep $count`_: Splits `$string` into a slice of substrings delimited by `$sep`, with number of substrings returned determined by `$count`. Alias for [`strings.SplitN`](https://golang.org/pkg/strings/#SplitN)
- _`sortStringsAsc $strings`_: Returns a slice of strings `$strings` sorted in ascending order.
- _`sortStringsDesc $strings`_: Returns a slice of strings `$strings` sorted in descending (reverse) order.
- _`sortObjectsByKeysAsc $objects $fieldPath`_: Returns the array `$objects`, sorted in ascending order based on the values of a field path expression `$fieldPath`.
- _`sortObjectsByKeysDesc $objects $fieldPath`_: Returns the array `$objects`, sorted in descending (reverse) order based on the values of a field path expression `$fieldPath`.
- _`trimPrefix $prefix $string`_: If `$prefix` is a prefix of `$string`, return `$string` with `$prefix` trimmed from the beginning. Otherwise, return `$string` unchanged.
- _`trimSuffix $suffix $string`_: If `$suffix` is a suffix of `$string`, return `$string` with `$suffix` trimmed from the end. Otherwise, return `$string` unchanged.
- _`toLower $string`_: Replace capital letters in `$string` to lowercase.
- _`toUpper $string`_: Replace lowercase letters in `$string` to uppercase.
- _`when $condition $trueValue $falseValue`_: Returns the `$trueValue` when the `$condition` is `true` and the `$falseValue` otherwise
- _`where $items $fieldPath $value`_: Filters an array or slice based on the values of a field path expression `$fieldPath`. A field path expression is a dot-delimited list of map keys or struct member names specifying the path from container to a nested value. Returns an array of items having that value.
- _`whereNot $items $fieldPath $value`_: Filters an array or slice based on the values of a field path expression `$fieldPath`. A field path expression is a dot-delimited list of map keys or struct member names specifying the path from container to a nested value. Returns an array of items **not** having that value.
- _`whereExist $items $fieldPath`_: Like `where`, but returns only items where `$fieldPath` exists (is not nil).
- _`whereNotExist $items $fieldPath`_: Like `where`, but returns only items where `$fieldPath` does not exist (is nil).
- _`whereAny $items $fieldPath $sep $values`_: Like `where`, but the string value specified by `$fieldPath` is first split by `$sep` into a list of strings. The comparison value is a string slice with possible matches. Returns items which OR intersect these values.
- _`whereAll $items $fieldPath $sep $values`_: Like `whereAny`, except all `$values` must exist in the `$fieldPath`.
- _`whereLabelExists $containers $label`_: Filters a slice of containers based on the existence of the label `$label`.
- _`whereLabelDoesNotExist $containers $label`_: Filters a slice of containers based on the non-existence of the label `$label`.
- _`whereLabelValueMatches $containers $label $pattern`_: Filters a slice of containers based on the existence of the label `$label` with values matching the regular expression `$pattern`.

---

### Examples

- [Automated Nginx Reverse Proxy for Docker](http://jasonwilder.com/blog/2014/03/25/automated-nginx-reverse-proxy-for-docker/)
- [Docker Log Management With Fluentd](http://jasonwilder.com/blog/2014/03/17/docker-log-management-using-fluentd/)
- [Docker Service Discovery Using Etcd and Haproxy](http://jasonwilder.com/blog/2014/07/15/docker-service-discovery/)

#### NGINX Reverse Proxy Config

[nginxproxy/nginx-proxy](https://hub.docker.com/r/nginxproxy/nginx-proxy) trusted build.

Start nginx-proxy:

```console
docker run -d -p 80:80 -v /var/run/docker.sock:/tmp/docker.sock -t nginxproxy/nginx-proxy
```

Then start containers with a VIRTUAL_HOST (and the VIRTUAL_PORT if more than one port is exposed) env variable:

```console
docker run -e VIRTUAL_HOST=foo.bar.com -e VIRTUAL_PORT=80 -t ...
```

If you wanted to run docker-gen directly on the host, you could do it with:

```console
docker-gen -only-published -watch -notify "/etc/init.d/nginx reload" templates/nginx.tmpl /etc/nginx/sites-enabled/default
```

#### Fluentd Log Management

This template generate a fluentd.conf file used by fluentd. It would then ship log files off
the host.

```console
docker-gen -watch -notify "restart fluentd" templates/fluentd.tmpl /etc/fluent/fluent.conf
```

#### Service Discovery in Etcd

This template is an example of generating a script that is then executed. This template generates
a python script that is then executed which register containers in Etcd using its HTTP API.

```console
docker-gen -notify "/bin/bash /tmp/etcd.sh" -interval 10 templates/etcd.tmpl /tmp/etcd.sh
```

### Development

This project uses [Go Modules](https://golang.org/ref/mod) for managing 3rd party dependencies.
This means that at least `go 1.11` is required.

For `go 1.11` and `go 1.12` it is additionally required to manually enable support by setting `GO111MODULE=on`.
For later versions, this is not required.

```console
git clone <your fork>
cd <your fork>
make get-deps
make
```

### License

MIT

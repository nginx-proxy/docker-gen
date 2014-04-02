docker-gen
=====

`docker-gen` is a config file generator that uses templates to generate files using docker container meta-data.

This is can be used to generate config files for:

 * fluentd, logstash or other centralized logging tools that tail the containers JSON log file.
 * logrotate files to rotate container JSON log files
 * nginx, haproxy, etc. reverse proxy configs to route requests from the host to containers

===

### Usage

`go get github.com/jwilder/docker-gen`

```
docker-gen
Usage: docker-gen [-config file] [-watch=false] [-notify="restart xyz"] <template> [<dest>]
```
  
*Options:*
* `-watch` - runs continuously and monitors docker container events.  When containers are started
or stopped, the template is regenerated.
* `-notify` - runs a command after the template is generated.  Useful for restarting nginx, reloading
haproxy, etc..
* `-config file` - Use the specified config file instead of command-line options.  Multiple templates can be defined and they will be executed in the order that they appear in the config file.

If no `<dest>` file is specified, the output is send to stdout.  Mainly useful for debugging.

### Examples

[Docker Log Management With Fluentd](http://jasonwilder.com/blog/2014/03/17/docker-log-management-using-fluentd/)

[Automated Nginx Reverse Proxy for Docker](http://jasonwilder.com/blog/2014/03/25/automated-nginx-reverse-proxy-for-docker/)


### TODO

 * Add a way to filter out containers in templates
 * Add a notify interval option

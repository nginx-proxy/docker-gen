docker-gen
=====

Config file generator using running docker container meta-data.

This is mostly a proof of concept to generate config files for:

 * fluentd, logstash or other centralized logging tools that tail the containers JSON log file.
 * logrotate files to rotate container JSON log files

[Docker Log Management With Fluentd](http://jasonwilder.com/blog/2014/03/17/docker-log-management-using-fluentd/)

===

To Run:

 `go get github.com/jwilder/docker-gen`

 `docker-gen template.file`

TODO:

 * Add restart command hooks for when files are regenerated.
 * Tail docker event stream to detect when containers are started and stopped automatically

package main

import (
	"bytes"
	"log"
	"os"
	"strings"

	"github.com/nginx-proxy/docker-gen/plugin"
)

var defaultHttpServer = `
server {
	listen 80 default_server;
	server_name _; # This is just an invalid value which will never trigger on a real hostname.
	error_log /proc/self/fd/2;
	access_log /proc/self/fd/1;
	return 503;
}
`

var upstreamServer = `
	# ${NAME}
	server ${IP}:${PORT};
`

var proxiedServer = `
upstream ${HOST} {
${UPSTREAMS}
}

server {
	gzip_types text/plain text/css application/json application/x-javascript text/xml application/xml application/xml+rss text/javascript;

	server_name ${HOST};
	proxy_buffering off;
	error_log /proc/self/fd/2;
	access_log /proc/self/fd/1;

	location / {
		proxy_pass ${HOST};
		proxy_set_header Host $http_host;
		proxy_set_header X-Real-IP $remote_addr;
		proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
		proxy_set_header X-Forwarded-Proto $scheme;

		# HTTP 1.1 support
		proxy_http_version 1.1;
		proxy_set_header Connection "";
	}
}
`

func virtualhost(t *plugin.RuntimeContainer) (*string, error) {
	if s, ok := t.Env["VIRTUAL_HOST"]; ok {
		return &s, nil
	}
	return nil, nil
}

func findAddressWithPort(c *plugin.RuntimeContainer, port string) bool {
	for _, address := range c.Addresses {
		if address.Port == port {
			return true
		}
	}
	return false
}

func NginxGen(in *plugin.PluginContext) (out []byte, err error) {
	bout := bytes.NewBuffer(make([]byte, 4096))
	bout.WriteString(defaultHttpServer)

	containersWithHosts, err := plugin.GroupByMulti(in.Containers, virtualhost, ",")
	if err != nil {
		return
	}
	for host, containers := range containersWithHosts {
		upstreams := make([]string, 0, len(containers))
		for _, c := range containers {
			if c.State.Health.Status != "" && c.State.Health.Status != "healthy" {
				log.Printf("Container %s is unhealthy", c.Name)
				continue
			}
			if len(c.Networks) == 0 {
				log.Printf("Container %s has no networks, but it is a virtual host %s", c.Name, host)
				continue
			}
			vars := map[string]string{"NAME": c.Name, "IP": c.Networks[0].IP, "PORT": ""}
			if len(c.Addresses) == 1 {
				// If only 1 port exposed, use that
				vars["PORT"] = c.Addresses[0].Port
			} else if port, ok := c.Env["VIRTUAL_PORT"]; ok {
				if findAddressWithPort(c, port) {
					vars["PORT"] = port
				} else {
					log.Printf("Container %s (vhost %s) has VIRTUAL_PORT %s, but it does not expose it", c.Name, host, port)
					continue
				}
			} else {
				// Else default to standard web port 80
				port := "80"
				if findAddressWithPort(c, port) {
					vars["PORT"] = port
				} else {
					log.Printf("Container %s (vhost %s) does not declare VIRTUAL_PORT, exposes multiple ports and does not expose port 80", c.Name, host)
					continue
				}
			}
			upstreams = append(upstreams, os.Expand(upstreamServer, func(s string) string { return vars[s] }))
		}
		expanded := os.Expand(proxiedServer, func(s string) string {
			switch s {
			case "HOST":
				return host
			case "UPSTREAMS":
				return strings.Join(upstreams, "\n")
			default:
				return "$" + s
			}
		})
		bout.WriteString(expanded)
	}

	out = bout.Bytes()
	return
}

func main() {
	plugin.Main(NginxGen)
}

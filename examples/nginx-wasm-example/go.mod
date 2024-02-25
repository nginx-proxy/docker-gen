module github.com/nginx-proxy/docker-gen/nginx-wasm-example

go 1.21

require github.com/nginx-proxy/docker-gen/plugin v0.0.0-00010101000000-000000000000

require (
	github.com/josharian/intern v1.0.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
)

replace github.com/nginx-proxy/docker-gen/plugin => ../../plugin

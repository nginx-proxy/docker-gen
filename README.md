docker-gen
=====

This is a direct fork of [nginx-proxy/docker-gen](https://github.com/nginx-proxy/docker-gen).

In addition to it's original features, it extends with the following:

- Container filter based on key/value for HUP signal
- Multiarch, with the following available in Docker hub:
  - amd64
  - arm64v8 (suited for Apple Sillicon) 
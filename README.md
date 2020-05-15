# unix-socket-http-veil

## Purpose

Allow filtering of requests against a UNIX domain socket by surfacing a new
UNIX domain socket that mantles specifically permitted HTTP request privileges
to the underlying API served by the socket

## Building

NOTE: For the greatest likelihood of compatibility, it is recommended to build
on the same CPU architecture that the intended target environment will use.

Please ensure that you have a working installation of Docker. Locate the
relevant instructions for your Operating System at
[the official Docker website](https://docs.docker.com/install).

```
docker pull golang:1.14.2-alpine
docker run -t -v $(pwd):/workenv -w /workenv golang:1.14.2-alpine go build src/veil.go
```

If the above commands are successful, an executable named `veil` should
appear.

## Usage

### Binary Executable

NOTE: Please be sure that the consumer can access the file descriptors at the
provided socket paths for the newly exposed UNIX domain sockets. Most likely
this will involve file-ownership and file-access privileges being consistent.

Invoke the executable as follows:

```
unix-socket-http-veil <path-to-target-socket> <path-to-exposed-api-socket> <path-to-access-rules-list>
```

Issue client requests against the new socket as follows -- cURL is only used as
an example, but any language ecosystem that supports communication with UNIX
domain sockets can be substituted here.

#### HTTP Request

```
curl -H "Content-Type: application/json" --unix-socket <path-to-exposed-api-socket> -X GET http://localhost/<http-request-path>
```
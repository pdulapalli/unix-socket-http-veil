# unix-socket-http-veil

## Purpose

Allow filtering of requests against a UNIX domain socket by surfacing a new
UNIX domain socket that mantles specific HTTP request privileges granted to the
underlying API served by the socket

## Building

NOTE: For the greatest likelihood of compatibility, it is recommended to build
on the same CPU architecture that the intended target environment will use.

Please ensure that you have a working installation of Docker. Locate the
relevant instructions for your Operating System at
[the official Docker website](https://docs.docker.com/install).


### Ubuntu/Debian Targets
```
docker pull golang:1.14.2-stretch
docker run -t -v $(pwd):/workenv -w /workenv golang:1.14.2-stretch go build src/veil.go
```

### Alpine Linux Targets
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

Issue client requests against the new, exposed socket as follows -- cURL is only used as
an example, but any language ecosystem that supports communication with UNIX
domain sockets can be substituted here.

The exposed socket will accept any requests but may refuse to respond to certain
requests if they are not specifically whitelisted in the
[access rules list](#access-rules-list).

#### HTTP Request

```
curl -H "Content-Type: application/json" --unix-socket <path-to-exposed-api-socket> -X GET http://localhost/<http-request-path>
```

### Access Rules List

An "access rules list" file must be provided to specify which HTTP request
method types and request paths are allowed against the veiled target UNIX
domain socket. If the file is empty, no HTTP requests to any paths will be
issued to the target socket.


#### Format

* Every allowance rule must be separated by a new line
* There can only be one allowance rule per line
* Each allowance rule must specify the HTTP Method and Request Path (relative to root)
  * The `~` character should be used to separate the HTTP Method and Request Path for each rule
* Only the following HTTP Methods are supported for allowance rule creation:
  * `GET`
  * `POST`
  * `DELETE`
  * `PATCH`
  * `PUT`

#### Example

An [example file](example/accessRulesList.txt.example) demonstrates the format
to expose HTTP `GET` methods against two different request paths.

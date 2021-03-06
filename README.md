# dcos-check-runner [![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0) [![Jenkins](https://jenkins.mesosphere.com/service/jenkins/buildStatus/icon?job=public-dcos-cluster-ops/dcos-check-runner/dcos-check-runner-master)](https://jenkins.mesosphere.com/service/jenkins/job/public-dcos-cluster-ops/job/dcos-check-runner/job/dcos-check-runner-master/) [![Go Report Card](https://goreportcard.com/badge/github.com/dcos/dcos-check-runner)](https://goreportcard.com/report/github.com/dcos/dcos-check-runner)

## DC/OS Check Runner
The DC/OS check runner is a utility that executes checks against a DC/OS node or cluster. The check runner reads check definitions from its configuration file and executes them when requested. Checks come in three types:

 * **node-prestart** checks, which assert that a host has prerequisites necessary for DC/OS to start
 * **node-poststart** checks, which assert that a DC/OS node is healthy
 * **cluster** checks, which assert that a DC/OS cluster is healthy

## Build
```
make build
./build/dcos-check-runner --version
```

## Test
```
make test
```

## Usage
```
  dcos-check-runner check <check-type> [flags]

Flags:
  -h, --help   help for check
      --list   List runner

Global Flags:
      --check-config string   Path to check configuration file (default "/opt/mesosphere/etc/dcos-check-config.json")
      --config string         config file (default is /opt/mesosphere/etc/dcos-check-runner.yaml)
      --role string           Set node role
      --verbose               Use verbose debug output.
      --version               Print dcos-check-runner version
```

```
  dcos-check-runner http-server [flags]

Flags:
      --base-uri string   Server's base URI
  -h, --help              help for http-server
  -a, --host string       Server's host (default "0.0.0.0")
  -p, --port int          Server's TCP port (default 8000)
      --systemd-socket    Listen on systemd socket

Global Flags:
      --check-config string   Path to check configuration file (default "/opt/mesosphere/etc/dcos-check-config.json")
      --config string         config file (default is /opt/mesosphere/etc/dcos-check-runner.yaml)
      --role string           Set node role
      --verbose               Use verbose debug output.
      --version               Print dcos-check-runner version
```

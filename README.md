# Docker PHP-FPM Wrapper

A solution to improve PHP application logging when running under php-fpm in a containerized environment. It also provides Prometheus monitoring and a graceful shutdown mechanism tailored for Kubernetes deployments.

## Overview

This project wraps around PHP-FPM, intercepting its logs and exposing Prometheus metrics. It is especially useful for containerized PHP applications running in Kubernetes, providing:

- **Enhanced Logging:** Routes php-fpm logs through a customizable logging proxy.
- **Slowlog Proxy:** Parse and proxy PHP-FPM slowlog output.
- **Logging Proxy:** Configure Unix pipe or socket for app log forwarding.
- **Prometheus Monitoring:** Exposes application metrics to help monitor application performance.
- **Graceful Shutdown:** Implements a delay mechanism to ensure smooth shutdowns in Kubernetes environments.

## Available Configuration Options

The following options can be configured via command-line flags or environment variables:

- **--log-level**
    - *Description:* Sets the internal log level. Use `-1` for debug-level logging.
    - *Default:* `-1`

- **--log-encoder**
    - *Description:* Chooses the internal logging encoder.
    - *Default:* `auto`

- **--fpm, -f**
    - *Description:* Path to the PHP-FPM executable.
    - *Default:* _(No default, must be provided if not in PATH)_

- **--fpm-config, -c**
    - *Description:* Path to the PHP-FPM configuration file.
    - *Default:* `/etc/php/php-fpm.conf`

- **--fpm-no-errlog**
    - *Description:* Disable php-fpm error log parsing and proxy.
    - *Default:* `false`

- **--fpm-no-slowlog**
    - *Description:* Disable php-fpm slow log parsing and proxy.
    - *Default:* `false`

- **--wrapper-pipe, -p**
    - *Description:* Path to the logging pipe. Set to an empty string (`""`) to disable.
    - *Default:* `/tmp/fpm-wrapper-pipe`

- **--wrapper-socket, -s**
    - *Description:* Path to the logging socket. Set to `null` to disable.
    - *Default:* `/tmp/fpm-wrapper.sock`

- **--line-buffer-size**
    - *Description:* Specifies the maximum log line size in bytes.
    - *Default:* `16384` (16*1024 bytes)

- **--listen**
    - *Description:* The address for Prometheus to scrape metrics.
    - *Default:* `:8080`

- **--metrics-path**
    - *Description:* The path to expose Prometheus metrics.
    - *Default:* `/metrics`

- **--shutdown-delay**
    - *Description:* Delay before the process shutdown to allow graceful termination.
    - *Default:* `500ms`

## Development

Run the checks before you push:

- `make lint` runs `golangci-lint`.
- `make test` runs the unit tests.
- `make e2e` runs the end-to-end tests. These tests need Docker.

The end-to-end tests start the wrapper with php-fpm in a container (testcontainers-go).
The default image is `php:8.5-fpm-alpine`. To use a different image, set `E2E_PHP_IMAGE`:

```sh
E2E_PHP_IMAGE=php:8.2-fpm make e2e
```

# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),  
and this project adheres to [Semantic Versioning](http://semver.org).

---

## [1.1.0] - 2026-10-21

### Added

- `--session-cleanup-interval` removes the expired PHP session files, as the `sessionclean` cron job of Debian does. The cleanup is off by default.

### Changed

- The minimum Go version is 1.27. Update all dependencies.
- The wrapper reads the php-fpm config with `php-fpm -tt` and runs php-fpm with an overlay config in `--run-dir`. The overlay moves `error_log` and the slowlogs to FIFOs in `--run-dir`. Before, the wrapper parsed the config itself and replaced the files at `error_log` and `slowlog` with FIFOs. These problems are gone:
  - The wrapper did not see `error_log` in an included file, so the error log of the official Docker images was not proxied.
  - The wrapper did not find a relative `include`, for example `include=etc/php-fpm.d/*.conf` of the PHP 7.4 images.
  - Without root, the wrapper could not create a FIFO for an `error_log` in `/var/log`.
- `error_log = syslog` also goes through the proxy. Use `--fpm-no-errlog` to keep it.
- The slowlog parser uses the `request_slowlog_trace_depth` of php-fpm (default 20). Before, the default was 64.
- When php-fpm rejects the config at start, the wrapper exits with the exit code of php-fpm, for example 78.
- Parse the flags and the environment variables with `go-flags` instead of `pflag` and `viper-lite`. The names of the flags and the environment variables do not change. These behaviors change:
  - An empty environment variable sets an empty value. For example, `WRAPPER_PIPE=` disables the pipe. Before, the wrapper ignored empty environment variables.
  - An argument before `--` that is not a flag of the wrapper stops the wrapper at start. Before, the wrapper ignored it. The wrapper passes the arguments after `--` to php-fpm, as before.
  - `--log-level` accepts a name or a number from `-1` to `5`. Other values stop the wrapper at start.
  - `--help` shows the environment variables and the defaults, and exits with code 0.
- The release binaries are static (`CGO_ENABLED=0`). The `linux/amd64` binary of 1.0.2 needed glibc and did not start on Alpine images.
- Replace the manual e2e test with automated e2e tests on testcontainers-go.
- Replace Travis CI with GitHub Actions: lint, unit tests, and e2e tests on PHP 5.6, 7.4, and 8.2–8.5.

### Fixed

- `--shutdown-delay` also applies to SIGQUIT. The php-fpm images have `STOPSIGNAL SIGQUIT`, so Docker and Kubernetes stopped php-fpm without the delay.
- The error log proxy writes a line in an unknown format to stderr without a change and continues. Before, it stopped at the first such line, and all next entries of the error log were lost. Such lines come, for example, from a pool with `decorate_workers_output = no`.
- The wrapper pipe (`FPM_WRAPPER_PIPE`) has mode `0666` for all umask values. Before, the umask gave `0644`, and workers that run as a different user than the master could not write to the pipe.
- The wrapper takes each section of the php-fpm config except `[global]` as a pool. Before, it ignored the pools without `pm.status_path`: their slowlog was not proxied. When such a pool was the last section, the wrapper did not start.
- The wrapper replaces each `$pool` in each directive of a pool, as php-fpm does. Before, it replaced only the first `$pool` in `slowlog`, so a `listen = /run/php/$pool.sock` pool had no metrics.
- The wrapper exits with an error when `--log-encoder` has an unknown value. Before, it panicked.

## [1.0.1] - 2025-01-10

### Fixed

- add psr-3 compatibility [pull request](https://github.com/code-tool/docker-fpm-wrapper/pull/4)
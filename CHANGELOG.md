# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),  
and this project adheres to [Semantic Versioning](http://semver.org).

---

## [Unreleased]

### Added

- `--session-cleanup-interval` removes the expired PHP session files, as the `sessionclean` cron job of Debian does. The cleanup is off by default.

### Changed

- The minimum Go version is 1.27. Update all dependencies.
- The release binaries are static (`CGO_ENABLED=0`). The `linux/amd64` binary of 1.0.2 needed glibc and did not start on Alpine images.
- Replace the manual e2e test with automated e2e tests on testcontainers-go.
- Replace Travis CI with GitHub Actions: lint, unit tests, and e2e tests on PHP 8.2–8.5.

### Fixed

- `--shutdown-delay` also applies to SIGQUIT. The php-fpm images have `STOPSIGNAL SIGQUIT`, so Docker and Kubernetes stopped php-fpm without the delay.
- The error log proxy writes a line in an unknown format to stderr without a change and continues. Before, it stopped at the first such line, and all next entries of the error log were lost. Such lines come, for example, from a pool with `decorate_workers_output = no`.
- The wrapper pipe (`FPM_WRAPPER_PIPE`) has mode `0666` for all umask values. Before, the umask gave `0644`, and workers that run as a different user than the master could not write to the pipe.
- The wrapper exits with an error when `--log-encoder` has an unknown value. Before, it panicked.

## [1.0.1] - 2025-01-10

### Fixed

- add psr-3 compatibility [pull request](https://github.com/code-tool/docker-fpm-wrapper/pull/4)
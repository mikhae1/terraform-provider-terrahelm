# v1.4.0

### New Features

- Add Helm 4 post-renderer support by generating a subprocess plugin in the TerraHelm cache
- Add exec-based Kubernetes authentication support
- Scope Helm cache/config/data/plugin directories to the TerraHelm cache
- Update Go version and dependencies and apply security fixes
- Add exec-based Kubernetes authentication support in provider configuration

### Bug Fixes

- Fix chart repository URL handling by using `--repo` for HTTP(S) repositories
- Fix download retry loop to run at least once when retries are disabled

# v1.3.3

### New Features

- Update goreleaser configuration to version 2

# v1.3.2

### New Features

- Update Terraform provider release action to version 4

# v1.3.1

### New Features

- Update go-getter to v1.7.8

# v1.3.0

### New Features

- Add max_retries and retry_delay options for remote downloads

# v1.2.2

### New Features

- Security fix

# v1.2.1

### New Features

- Update documentation

# v1.2.0

### New Features

- Add support for `post_renderer` option in terrahelm release
- Add support for `post_renderer_url` option in terrahelm release

# v1.1.1

### Bug Fixes

- Remove unused values from data source
- Update documentation

# v1.1.0

### New Features

- Add support for `chart_url` option in terrahelm release for downloading charts from Git, Mercurial, HTTP, Amazon S3, Google GCP and local filesystem
- Add support for `insecure` option in terrahelm release
- Add support for `git_bin_path` option in provider configuration

### Breaking changes

- Rename `helm_repository` option to `chart_repository`

# v1.0.3

### New Features

- Add support for passing custom arguments to helm commands via the `custom_args` attribute
- Add support for `debug` attribute in terrahelm release

### Bug Fixes

- Fix always remove directory before git clone

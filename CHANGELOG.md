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

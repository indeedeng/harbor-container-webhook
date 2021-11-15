# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.4] - 2021-11-15
### Added
- New prometheus metric `hcw.mutations.image_rewrite` which tracks the original and rewritten image modified

## [0.3.2] - 2021-10-05
### Fixed
- Chart handling of `rules` and `extraRules` was incorrect when unset.

## [0.3.0] - 2021-10-04
### Changed
- Rewrote significant parts of the implementation and configuration to switch to a new regex based rules system.
### Fixed
- Chart version supports cloud vendor prelease suffixes

## [0.2.0] - 2020-10-29
### Changed
- Rewrote webhook to use containers/image reference parsing instead of regex
### Added
- Added verbose mode flag

## [0.1.0] - 2020-10-19
### Added
- Initial release

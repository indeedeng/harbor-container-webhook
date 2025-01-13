# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.7.1] - 2025-01-13
### Changed
- Updated downstream libraries, go version, etc

## [0.7.0] - 2024-01-31
### Changed
- Instead of querying for the node architecture and os when inspecting pods, which rarely worked, use `platforms` on the config to determine which platforms should be required when checking upstream.

## [0.6.3] - 2024-01-26
### Fixed
- Fixed rewrite_success prometheus metric counting every rule invocation, instead of only rewrites
### Changed
- Added helm usage to the README.md

## [0.6.2] - 2023-12-06
### Fixed
- Fixed ServiceMonitor templates in helm chart not rendering correctly (thanks @z0rc for the fix!)
### Changed
- Updated go.mod dependencies

## [0.6.1] - 2023-12-04
### Added
- Added cluster role permissions for list, watch on nodes

## [0.6.0] - 2023-12-04
### Added
- Support for authenticating to check if manifests exist for each proxy rule with an image pull secret.
### Changed
- Changes to the helm chart RBAC to support access secrets within the webhook's namespace.
- Some minor test refactoring.
- Deprecated kube-client-lazy-remap flag (no-op now), it has graduated to default controller runtime behavior

## [0.5.0] - 2023-05-12
### Added
- Added cli flags for passing kube client qps, burst, and enabling lazy rest mapping of resources in the controller-runtime

## [0.4.2] - 2023-04-24
### Changed
- Changed node lookup for pod submissions to fail-open and default to webhook's OS and architecture

## [0.4.1] - 2023-03-28
### Fixed
- Fixed node lookup with untyped client, did not pass struct pointer correctly
### Changed
- Improved logging around rejected pod submissions due to node lookup.

## [0.4.0] - 2023-03-23
### Added
- Added detection of the pod OS, architecture for manifests
- Added cluster role and bindings for accessing node resources
### Changed
- Rebuilt and upgraded modules

## [0.3.5] - 2022-10-17
### Changed
- Added volumes, volume mounts, init containers to the helm chart
- Rebuilt and upgraded modules, other minor tlc.

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

# Changelog

All notable changes to **`many_faces_push`** are documented here.

Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) — **version headings only, no dates**. SemVer: [`VERSION`](./VERSION).

### Release index

| Version       | Theme                                  |
| ------------- | -------------------------------------- |
| [0.4.0](#040) | Per-request FCM and TestFcmCredentials |
| [0.3.0](#030) | TLS/mTLS and many_faces_proto          |
| [0.2.0](#020) | gRPC PushService and FCM SendPush      |
| [0.1.0](#010) | Worker skeleton                        |

## [Unreleased]

### Added

### Changed

### Fixed

---

## [0.4.0]

### Added

- Per-request FCM credentials override; TestFcmCredentials RPC; verify-edge-contracts; lint.sh.

## [0.3.0]

### Added

- gRPC TLS and optional mTLS; nested many_faces_proto submodule.

### Fixed

- TLS smoke grpcurl key permissions; vendored health.proto for grpcurl.

## [0.2.0]

### Added

- gRPC PushService with FCM SendPush; proto v1; start-push-worker script.

## [0.1.0]

### Added

- Push worker skeleton with Docker compose, Dockerfile, and CI.

[Unreleased]: https://github.com/01laky/many_faces_push/compare/v0.4.0...HEAD
[0.4.0]: https://github.com/01laky/many_faces_push/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/01laky/many_faces_push/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/01laky/many_faces_push/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/01laky/many_faces_push/releases/tag/v0.1.0

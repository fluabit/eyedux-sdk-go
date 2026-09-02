# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.4.0] - 2026-09-02

### Added

- `EventEyeduxType` with predefined system event categories.
- `NewWithConfig` with required `APIKey` and `ProjectID` fields.
- `NewFromEnv`, which reads only `EYEDUX_API_KEY`.
- `WithProjectID`, `WithDefaultMetadata`, and `IsExternalObjectConflict`.
- `ErrEmptyProjectID` for missing project configuration.

### Changed

- `Event.EyeduxType` now uses `*EventEyeduxType`.
- `CreateEventInput.EyeduxType` now uses `EventEyeduxType`; `ProjectID` remains
	available until a future major version.

## [0.3.3] - 2026-09-02

### Added

- `EyeduxType` can be sent when creating an event.
- `Environment` and `EyeduxType` are available on returned events.

## [0.3.2] - 2026-08-27

### Added

- `TypeGroup` field added to `Event` and `CreateEventInput` for enhanced event categorization.

## [0.3.1] - 2026-08-19

### Added

- CHANGELOG.md.

## [0.3.0] - 2026-08-19

### Changed

- `external_object` and `correlation_object` are now structured `EventObject` values (`id`, `property`, `source`) instead of plain strings, matching the updated API contract.
- `Event.ExternalID *string` → `Event.ExternalObject *EventObject`
- `Event.CorrelationID *string` → `Event.CorrelationObject *EventObject`
- Same field replacements in `CreateEventInput`.
- Error code `event_external_id_conflict` renamed to `event_external_object_conflict`.
- Constant `ErrCodeEventExternalIDConflict` renamed to `ErrCodeEventExternalObjectConflict`.

## [0.2.0] - 2026-08-19

### Changed

- Package renamed from `eyedux` to `eyeduxsdk` to avoid import path conflicts.

## [0.1.1] - 2026-08-19

### Changed

- `CreateEvent` refactored to use an explicit `createEventBody` constructor for better readability.

## [0.1.0] - 2026-08-19

### Added

- Initial release.
- `Client` with `New`, `CreateEvent`, `ListEvents`, and `FindEventByExternalID`.
- `Event`, `CreateEventInput`, `ListEventsInput` types.
- Structured API error handling (`APIError`, `IsConflict`, `IsNotFound`, `IsAuthError`).
- `ErrEmptyAPIKey`, `ErrEmptyExternalID` sentinel errors.

[Unreleased]: https://github.com/fluabit/eyedux-sdk-go/compare/v0.4.0...HEAD
[0.4.0]: https://github.com/fluabit/eyedux-sdk-go/compare/v0.3.3...v0.4.0
[0.3.3]: https://github.com/fluabit/eyedux-sdk-go/compare/v0.3.2...v0.3.3
[0.3.2]: https://github.com/fluabit/eyedux-sdk-go/compare/v0.3.1...v0.3.2
[0.3.1]: https://github.com/fluabit/eyedux-sdk-go/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/fluabit/eyedux-sdk-go/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/fluabit/eyedux-sdk-go/compare/v0.1.1...v0.2.0
[0.1.1]: https://github.com/fluabit/eyedux-sdk-go/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/fluabit/eyedux-sdk-go/releases/tag/v0.1.0

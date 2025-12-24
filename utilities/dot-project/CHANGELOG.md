# Changelog

All notable changes to the Project Validator will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.0] - 2024-12-25

### Added
- **Extension mechanism for third-party tools** (addresses issue #125)
  - New `extensions` field in project YAML schema
  - Support for tool-specific configuration with versioning
  - Extension metadata including author, homepage, repository, and license
  - Comprehensive validation for extension names and structure
- **Experimental fields support**
  - New `experimental` field for testing new functionality
  - Flexible structure for prototype features
- **Enhanced validation**
  - Extension name format validation (alphanumeric, hyphens, underscores, dots)
  - Reserved name checking to prevent conflicts with core fields
  - Metadata URL validation for extensions
  - Comprehensive test coverage for new features
- **Documentation and examples**
  - Updated README with extension system documentation
  - Migration guide for upgrading to schema version 1.1.0
  - Example project demonstrating extension usage
  - Comprehensive test cases for extension validation

### Changed
- **Schema version updated to 1.1.0** to support extensions
- **Project struct enhanced** with Extensions and Experimental fields
- **Validation logic extended** to include extension validation
- **Test project updated** to demonstrate extension capabilities

### Technical Details
- Added `Extension` and `ExtensionMetadata` types
- Implemented `validateExtensions()` function with comprehensive checks
- Added `isValidExtensionName()` and `isReservedExtensionName()` helpers
- Enhanced project validation to include extension validation
- Maintained full backward compatibility with existing projects

### Files Added
- `extensions_test.go` - Comprehensive test suite for extension validation
- `yaml/example-with-extensions.yaml` - Example project with extensions
- `MIGRATION.md` - Migration guide for extension support
- `CHANGELOG.md` - This changelog file

### Files Modified
- `types.go` - Added Extension and ExtensionMetadata types
- `validator.go` - Enhanced validation with extension support
- `README.md` - Updated documentation with extension system
- `yaml/test-project.yaml` - Updated with extension examples
- `yaml/projectlist.yaml` - Added example project to validation list

## [1.0.0] - Previous Release

### Features
- Project YAML validation against structured schema
- Content drift detection using SHA256 hashes
- Maintainer validation against canonical sources
- Multiple output formats (text, JSON, YAML)
- GitHub Actions workflow integration
- Comprehensive test suite

---

## Extension System Overview

The extension mechanism introduced in v1.1.0 provides:

1. **Namespaced Configuration**: Tools can store configuration without conflicts
2. **Version Tracking**: Each extension includes version information
3. **Metadata Support**: Rich metadata for tool discovery and documentation
4. **Validation**: Comprehensive validation ensures data integrity
5. **Backward Compatibility**: Existing projects continue to work unchanged

## Migration Path

Existing projects can adopt extensions by:
1. Updating `schema_version` to "1.1.0"
2. Adding `extensions` section for tool configurations
3. Using `experimental` for testing new features
4. Running validation to ensure correctness

This addresses the community need expressed in issue #125 for extensible project configuration while maintaining the integrity and compatibility of the core specification.
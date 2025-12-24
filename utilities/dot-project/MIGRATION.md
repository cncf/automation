# Migration Guide: Extension Support

This guide helps you migrate existing `.project` files to support the new extension mechanism introduced in schema version 1.1.0.

## Overview

The extension mechanism allows third-party tools to store their configuration alongside core project metadata without conflicts. This addresses issue #125 by providing a structured way for tools to extend the `.project` format.

## Schema Version Update

Update your `schema_version` to enable extension support:

```yaml
# Before
schema_version: "1.0.0"

# After  
schema_version: "1.1.0"
```

## Backward Compatibility

- **Existing projects continue to work** - no breaking changes to core fields
- **Optional fields** - extensions and experimental sections are completely optional
- **Validation** - existing validation rules remain unchanged

## Adding Extensions

### Basic Extension

```yaml
extensions:
  my-tool:
    version: "1.0.0"
    config:
      enabled: true
      setting: "value"
```

### Full Extension with Metadata

```yaml
extensions:
  security-scanner:
    version: "2.1.0"
    description: "Automated security scanning"
    config:
      scan_schedule: "daily"
      exclude_paths: ["vendor/", "test/"]
    metadata:
      author: "Security Team"
      homepage: "https://security-scanner.example.com"
      repository: "https://github.com/security/scanner"
      license: "Apache-2.0"
```

## Extension Naming Rules

✅ **Valid Names:**
- `security-scanner`
- `deployment_tool`
- `monitoring.agent`
- `MyTool123`

❌ **Invalid Names:**
- `invalid@name` (special characters)
- `invalid name` (spaces)
- `cncf` (reserved name)
- `core` (reserved name)

## Reserved Names

The following names are reserved and cannot be used for extensions:
- Core fields: `name`, `description`, `repositories`, etc.
- CNCF-specific: `cncf`, `kubernetes`
- System: `core`, `system`, `extensions`, `experimental`

## Experimental Fields

For testing new functionality that doesn't fit the extension model:

```yaml
experimental:
  custom_feature:
    enabled: true
    config: "value"
  
  beta_api: "v2"
```

## Migration Examples

### Before (Schema 1.0.0)
```yaml
name: "My Project"
schema_version: "1.0.0"
# ... core fields
```

### After (Schema 1.1.0)
```yaml
name: "My Project"
schema_version: "1.1.0"
# ... core fields

extensions:
  ci-tool:
    version: "1.0.0"
    config:
      auto_deploy: true
      environments: ["staging", "prod"]

experimental:
  new_feature: "testing"
```

## Validation Changes

The validator now checks:
- Extension name format and reserved names
- Required version field for extensions
- Metadata URL validation (if provided)
- Experimental field naming conventions

## Tool Integration

### For Tool Authors

1. **Choose a unique name** following naming conventions
2. **Version your extension** for compatibility tracking
3. **Provide metadata** for discoverability
4. **Document your configuration** schema

### For Project Maintainers

1. **Update schema version** to 1.1.0
2. **Add tool configurations** under `extensions`
3. **Use experimental** for testing new features
4. **Validate regularly** to catch issues early

## Common Issues

### Extension Name Conflicts
```yaml
# Problem: Reserved name
extensions:
  core:  # ❌ Reserved
    version: "1.0.0"

# Solution: Use descriptive name
extensions:
  my-core-tool:  # ✅ Valid
    version: "1.0.0"
```

### Missing Version
```yaml
# Problem: No version
extensions:
  tool:
    config: {}  # ❌ Missing version

# Solution: Always include version
extensions:
  tool:
    version: "1.0.0"  # ✅ Required
    config: {}
```

### Invalid URLs in Metadata
```yaml
# Problem: Invalid URL
extensions:
  tool:
    version: "1.0.0"
    metadata:
      homepage: "not-a-url"  # ❌ Invalid

# Solution: Use valid URLs
extensions:
  tool:
    version: "1.0.0"
    metadata:
      homepage: "https://tool.example.com"  # ✅ Valid
```

## Testing Your Migration

1. **Update schema version** to 1.1.0
2. **Add your extensions**
3. **Run the validator**: `./validator`
4. **Fix any validation errors**
5. **Test with your tools**

## Support

For questions about the extension mechanism:
- Review the [README.md](README.md) for detailed documentation
- Check existing examples in `yaml/example-with-extensions.yaml`
- Refer to issue #125 for background and discussion
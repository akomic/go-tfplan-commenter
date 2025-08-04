# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2025-08-04

### Added
- Initial release of tfplan-commenter CLI tool
- Parse Terraform plan JSON files and generate Markdown comments
- Support for Create, Update, Delete, and Replace operations
- Detailed attribute change analysis for updated resources
- Replacement reason detection for replaced resources
- Resource deletion context for deleted resources
- Cross-platform binary releases (Linux x86_64, macOS Intel, macOS Apple Silicon)
- Version information and help commands
- GitHub Actions workflows for CI/CD and releases
- Comprehensive documentation and examples

### Features
- **Detailed Attribute Changes**: Shows exactly which attributes are being modified with before/after values
- **Replacement Analysis**: Automatically detects and explains why resources are being replaced
- **Smart Filtering**: Skips system-generated attributes that aren't meaningful to users
- **Enhanced Formatting**: Clean, readable Markdown output with emojis and proper formatting
- **Multi-platform Support**: Pre-built binaries for Linux and macOS (Intel & Apple Silicon)
- **Version Management**: Built-in version information with git commit and build date

[Unreleased]: https://github.com/akomic/go-tfplan-commenter/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/akomic/go-tfplan-commenter/releases/tag/v1.0.0

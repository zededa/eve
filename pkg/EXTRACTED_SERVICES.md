# Extracted Services Build Configuration

This directory contains services that have been extracted from the pillar monolith
for improved security, maintainability, and deployment flexibility.

## Services

### hardware-model
- **Status**: ✅ Complete and Integrated
- **Purpose**: Hardware model detection service
- **Location**: `pkg/hardware-model/`
- **Benefits**: Reduced attack surface, independent deployment

### Next Candidates
- `command` - Test/debug utility service
- `client` - Device registration service (complex)
- `diag` - Diagnostic utilities service

## Build Instructions

Each extracted service can be built independently:

```bash
# Build hardware-model service
cd pkg/hardware-model
make build

# Build Docker image
make build-docker

# Run tests
make test
```

## Integration Status

- [x] hardware-model: Extracted and validated
- [ ] command: Ready for extraction
- [ ] Integration with main EVE build system
- [ ] Container orchestration updates
- [ ] Documentation updates

## Migration Strategy

1. **Phase 1**: Extract simple, independent services
2. **Phase 2**: Update build system integration
3. **Phase 3**: Migrate pillar consumers to use extracted services
4. **Phase 4**: Remove extracted services from pillar monolith
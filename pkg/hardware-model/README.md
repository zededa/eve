# Hardware Model Service

This package contains the extracted hardware model detection service, separated from the pillar monolith for better maintainability and security.

## Purpose

The hardware model service provides system hardware model information in a simple, standalone format. It reads hardware information and outputs the model string.

## Usage

This service can be run standalone or as part of the EVE system:

```bash
# Run standalone
./hardware-model --o /tmp/output.txt

# Without newline
./hardware-model --c --o /tmp/output.txt
```

## Options

- `--c`: No CRLF (carriage return/line feed)
- `--o`: Output file or device (default: /dev/tty)

## Dependencies

- `github.com/lf-edge/eve/pkg/pillar/hardware` - for hardware detection
- `github.com/lf-edge/eve/pkg/pillar/base` - for logging

## Security Benefits

By extracting this service from the pillar monolith:
- Reduced attack surface
- Clear service boundaries
- Independent security updates
- Easier testing and validation
# Command Service

This package contains the extracted command execution service, separated from the pillar monolith for better maintainability and security.

## Purpose

The command service provides a simple interface for executing system commands with timeout, environment variable support, and output handling. It's primarily used for testing and debugging.

## Usage

```bash
# Interactive mode - reads commands from stdin
echo 'ls -l /tmp' | ./command-service

# With timeout
echo 'sleep 10' | ./command-service -t 5

# With environment variable
echo 'echo $TEST_VAR' | ./command-service -e TEST_VAR=hello

# Combined stdout/stderr
echo 'ls /nonexistent' | ./command-service -c

# Quiet mode
echo 'date' | ./command-service -q

# Don't wait for completion
echo 'sleep 5' | ./command-service -W
```

## Options

- `-t <seconds>`: Maximum time to wait for command (default: 200)
- `-c`: Combine stdout and stderr
- `-e <name=value>`: Set environment variable
- `-q`: Quiet mode (reduce logging)
- `-W`: Don't wait for result

## Dependencies

- `github.com/lf-edge/eve/pkg/pillar/execlib` - for command execution
- `github.com/lf-edge/eve/pkg/pillar/base` - for logging

## Security Benefits

By extracting this service from the pillar monolith:
- Isolated command execution environment
- Reduced attack surface on main pillar
- Independent security updates
- Clear service boundaries for audit
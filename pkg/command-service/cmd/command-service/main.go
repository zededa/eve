// Copyright (c) 2020 Zededa, Inc.
// SPDX-License-Identifier: Apache-2.0

// Command Service - Standalone command execution service
// Extracted from pillar monolith for improved security and maintainability

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/lf-edge/eve/pkg/pillar/agentbase"
	"github.com/lf-edge/eve/pkg/pillar/base"
	"github.com/lf-edge/eve/pkg/pillar/execlib"
	"github.com/lf-edge/eve/pkg/pillar/pubsub"
	"github.com/lf-edge/eve/pkg/pillar/pubsub/socketdriver"
	"github.com/sirupsen/logrus"
)

const (
	serviceName = "command-service"
	version     = "1.0.0"
)

// ServiceState holds the configuration for the command service
type ServiceState struct {
	agentbase.AgentBase
	quietPtr     *bool
	timeLimitPtr *uint
	combinedPtr  *bool
	environPtr   *string
	dontWaitPtr  *bool
	showHelp     *bool
	showVersion  *bool
}

// AddServiceCLIFlags adds command line flags specific to this service
func (state *ServiceState) AddAgentSpecificCLIFlags(flagSet *flag.FlagSet) {
	state.quietPtr = flagSet.Bool("q", false, "Quiet mode (reduce logging)")
	state.timeLimitPtr = flagSet.Uint("t", 200, "Maximum time to wait for command")
	state.combinedPtr = flagSet.Bool("c", false, "Combine stdout and stderr")
	state.environPtr = flagSet.String("e", "", "Set environment variable with name=val syntax")
	state.dontWaitPtr = flagSet.Bool("W", false, "Don't wait for result")
	state.showHelp = flagSet.Bool("h", false, "Show help")
	state.showVersion = flagSet.Bool("version", false, "Show version")
}

func main() {
	// Initialize logging
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	
	// Report nano timestamps
	formatter := logrus.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
	}
	logger.SetFormatter(&formatter)
	
	log := base.NewSourceLogObject(logger, serviceName, 0)

	// Create service state
	state := ServiceState{}
	
	// Initialize agent base with flags
	agentbase.Init(&state, logger, log, serviceName,
		agentbase.WithArguments(os.Args[1:]))

	// Handle help and version flags
	if *state.showHelp {
		printUsage()
		os.Exit(0)
	}

	if *state.showVersion {
		fmt.Printf("%s version %s\n", serviceName, version)
		os.Exit(0)
	}

	// Apply quiet mode
	if *state.quietPtr {
		logger.SetLevel(logrus.WarnLevel)
	}

	// Set up environment
	var environ []string
	if *state.environPtr != "" {
		environ = append(environ, *state.environPtr)
	}

	// Initialize pubsub for execlib
	ps := pubsub.New(&socketdriver.SocketDriver{
		Logger: logger,
		Log:    log,
	}, logger, log)

	// Initialize executor
	hdl, err := execlib.New(ps, log, serviceName, "executor")
	if err != nil {
		log.Fatal("Failed to initialize executor", err)
	}

	// Read and execute commands from stdin
	r := bufio.NewReader(os.Stdin)
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				fmt.Fprintf(os.Stderr, "Read failed: %s\n", err)
			}
			break
		}
		
		tokens := strings.Split(strings.TrimSpace(line), " ")
		if len(tokens) == 0 {
			continue
		}
		
		execute(hdl, tokens[0], tokens[1:], environ, *state.timeLimitPtr, *state.combinedPtr, *state.dontWaitPtr)
	}
}

func execute(hdl *execlib.ExecuteHandle, command string, args []string, environ []string, timeLimit uint, combinedOutput bool, dontWait bool) {
	out, err := hdl.Execute(execlib.ExecuteArgs{
		Command:        command,
		Args:           args,
		Environ:        environ,
		TimeLimit:      timeLimit,
		CombinedOutput: combinedOutput,
		DontWait:       dontWait,
	})
	
	if err != nil {
		fmt.Printf("Failed: %s\n", err)
		fmt.Printf("Failed output: %s\n", out)
		return
	}
	
	if dontWait {
		fmt.Printf("requested DontWait: no output\n")
	} else {
		fmt.Printf("Output:\n%s\n", out)
	}
}

func printUsage() {
	fmt.Printf(`%s - Command Execution Service

USAGE:
    %s [OPTIONS]

OPTIONS:
    -t <seconds>     Maximum time to wait for command (default: 200)
    -c               Combine stdout and stderr
    -e <name=value>  Set environment variable
    -q               Quiet mode (reduce logging)
    -W               Don't wait for result
    -h               Show this help message
    --version        Show version information

DESCRIPTION:
    This service executes system commands read from stdin with timeout,
    environment variable support, and output handling. It's designed as
    a standalone service extracted from the EVE pillar monolith for
    improved security and maintainability.

EXAMPLES:
    # Execute a simple command
    echo 'ls -l /tmp' | %s
    
    # With timeout
    echo 'sleep 10' | %s -t 5
    
    # With environment variable
    echo 'echo $TEST_VAR' | %s -e TEST_VAR=hello
    
    # Combined output
    echo 'ls /nonexistent' | %s -c
    
    # Don't wait for completion
    echo 'sleep 5' | %s -W

INPUT FORMAT:
    Commands are read from stdin, one per line. Each line should contain
    the command and its arguments separated by spaces.

`, serviceName, serviceName, serviceName, serviceName, serviceName, serviceName, serviceName)
}
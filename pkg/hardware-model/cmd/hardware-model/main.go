// Copyright (c) 2018 Zededa, Inc.
// SPDX-License-Identifier: Apache-2.0

// Hardware Model Service - Standalone hardware model detection service
// Extracted from pillar monolith for improved security and maintainability

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/lf-edge/eve/pkg/pillar/agentbase"
	"github.com/lf-edge/eve/pkg/pillar/base"
	"github.com/lf-edge/eve/pkg/pillar/hardware"
	"github.com/sirupsen/logrus"
)

const (
	serviceName = "hardware-model"
	version     = "1.0.0"
)

// ServiceState holds the configuration for the hardware model service
type ServiceState struct {
	agentbase.AgentBase
	noCRLF     *bool
	outputFile *string
	showHelp   *bool
	showVersion *bool
}

// AddServiceCLIFlags adds command line flags specific to this service
func (state *ServiceState) AddAgentSpecificCLIFlags(flagSet *flag.FlagSet) {
	state.noCRLF = flagSet.Bool("c", false, "No CRLF (carriage return/line feed)")
	state.outputFile = flagSet.String("o", "/dev/tty", "Output file or device")
	state.showHelp = flagSet.Bool("h", false, "Show help")
	state.showVersion = flagSet.Bool("version", false, "Show version")
}

func main() {
	// Initialize logging
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
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

	// Get hardware model
	model := hardware.GetHardwareModelNoOverride(log)
	if model == "" {
		log.Fatal("Failed to detect hardware model")
	}

	// Prepare output  
	var output string
	if *state.noCRLF {
		output = model
	} else {
		output = model + "\n"
	}

	// Write to output file
	err := os.WriteFile(*state.outputFile, []byte(output), 0644)
	if err != nil {
		log.Fatal("Failed to write to output file", err, *state.outputFile)
	}
}

func printUsage() {
	fmt.Printf(`%s - Hardware Model Detection Service

USAGE:
    %s [OPTIONS]

OPTIONS:
    -c              No CRLF (carriage return/line feed)
    -o <file>       Output file or device (default: /dev/tty)
    -h              Show this help message
    --version       Show version information

DESCRIPTION:
    This service detects and outputs the hardware model of the current system.
    It's designed as a standalone service extracted from the EVE pillar monolith
    for improved security and maintainability.

EXAMPLES:
    # Output to terminal
    %s

    # Output to file without newline
    %s -c -o /tmp/hardware-model.txt

    # Output to file with newline
    %s -o /tmp/hardware-model.txt

`, serviceName, serviceName, serviceName, serviceName, serviceName)
}
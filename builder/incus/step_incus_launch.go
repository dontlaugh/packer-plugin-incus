// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package incus

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type stepIncusLaunch struct{}

func (s *stepIncusLaunch) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packersdk.Ui)

	name := config.ContainerName
	image := config.Image
	profile := fmt.Sprintf("--profile=%s", config.Profile)
	vm := fmt.Sprintf("--vm=%s", strconv.FormatBool(config.VirtualMachine))

	launch_args := []string{
		"launch", "--ephemeral=false", vm, profile, image, name,
	}

	for k, v := range config.LaunchConfig {
		launch_args = append(launch_args, "--config", fmt.Sprintf("%s=%s", k, v))
	}

	ui.Say("Creating container...")
	_, err := IncusCommand(launch_args...)
	if err != nil {
		err := fmt.Errorf("Error creating container: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}
	sleepSeconds, err := strconv.Atoi(config.InitSleep)
	if err != nil {
		err := fmt.Errorf("Error parsing InitSleep into int: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	// TODO: Should we check `incus info <container>` for "Running"?
	// We have to do this so /tmp doesn't get cleared and lose our provisioner scripts.

	time.Sleep(time.Duration(sleepSeconds) * time.Second)
	log.Printf("Sleeping for %d seconds...", sleepSeconds)
	return multistep.ActionContinue
}

func (s *stepIncusLaunch) Cleanup(state multistep.StateBag) {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packersdk.Ui)

	if config.SkipPublish {
		return
	}

	cleanupArgs := []string{
		"delete", "--force", config.ContainerName,
	}

	ui.Say("Unregistering and deleting deleting container...")
	if _, err := IncusCommand(cleanupArgs...); err != nil {
		ui.Error(fmt.Sprintf("Error deleting container: %s", err))
	}
}

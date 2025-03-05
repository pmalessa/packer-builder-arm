// step_map_image.go
package builder

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
)

// StepMapImage maps a system image to a free loop device and creates partition mappings via kpartx.
type StepMapImage struct {
	ResultKey  string
	loopDevice string
}

// Run the step.
func (s *StepMapImage) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	config := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)
	image := config.ImageConfig.ImagePath

	ui.Message(fmt.Sprintf("Mapping image %s to free loopback device with partition mapping", image))

	// Map the image using losetup with partition scanning enabled.
	out, err := exec.Command("losetup", "--find", "--show", "--partscan", image).CombinedOutput()
	if err != nil {
		ui.Error(fmt.Sprintf("Error running losetup: %v: %s", err, string(out)))
		return multistep.ActionHalt
	}
	s.loopDevice = strings.TrimSpace(string(out))
	ui.Message(fmt.Sprintf("Image %s mapped to %s", image, s.loopDevice))

	// Use kpartx to create device-mapper entries for the partitions.
	out, err = exec.Command("kpartx", "-av", s.loopDevice).CombinedOutput()
	if err != nil {
		ui.Error(fmt.Sprintf("Error running kpartx: %v: %s", err, string(out)))
		return multistep.ActionHalt
	}
	ui.Message(fmt.Sprintf("kpartx mapping output:\n%s", string(out)))

	// Convert the loop device to the device-mapper path.
	// For example, if s.loopDevice is "/dev/loop0", mappedDevice becomes "/dev/mapper/loop0"
	mappedDevice := "/dev/mapper/" + strings.TrimPrefix(s.loopDevice, "/dev/")
	ui.Message(fmt.Sprintf("Returning mapped device: %s", mappedDevice))

	// Store the mapped device in state for later steps.
	state.Put(s.ResultKey, mappedDevice)
	return multistep.ActionContinue
}


// Cleanup removes kpartx mappings and detaches the loop device.
func (s *StepMapImage) Cleanup(state multistep.StateBag) {
	ui := state.Get("ui").(packer.Ui)
	if s.loopDevice == "" {
		return
	}
	// Remove kpartx mappings.
	out, err := exec.Command("kpartx", "-d", s.loopDevice).CombinedOutput()
	if err != nil {
		ui.Error(fmt.Sprintf("Error cleaning up kpartx mappings for %s: %v: %s", s.loopDevice, err, string(out)))
	}
	// Detach the loop device.
	out, err = exec.Command("losetup", "--detach", s.loopDevice).CombinedOutput()
	if err != nil {
		ui.Error(fmt.Sprintf("Error detaching loop device %s: %v: %s", s.loopDevice, err, string(out)))
	}
}

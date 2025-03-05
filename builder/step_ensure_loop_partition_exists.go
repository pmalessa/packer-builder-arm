// StepEnsureLoopPartitionExists ensures that the loop partition device exists.
// It reads the loop device and partition number from state and creates the missing
// device node if necessary.
package builder

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
)

// StepEnsureLoopPartitionExists creates missing loop partition device nodes.
type StepEnsureLoopPartitionExists struct {
	// LoopDeviceKey is the state key where the parent loop device is stored.
	LoopDeviceKey string
	// PartitionKey is the state key where the partition number is stored.
	PartitionKey string
}

// Run the step.
func (s *StepEnsureLoopPartitionExists) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packer.Ui)

	loopDevice, ok := state.Get(s.LoopDeviceKey).(string)
	if !ok || loopDevice == "" {
		ui.Error("no loop device found in state")
		return multistep.ActionHalt
	}

	partition, ok := state.Get(s.PartitionKey).(int)
	if !ok || partition <= 0 {
		ui.Error("invalid partition number in state")
		return multistep.ActionHalt
	}

	// Expected child device, e.g. "/dev/loop0p2"
	childDevice := fmt.Sprintf("%sp%d", loopDevice, partition)
	ui.Message(fmt.Sprintf("ensuring partition device %s exists", childDevice))

	// If the device node already exists, nothing to do.
	if _, err := os.Stat(childDevice); err == nil {
		ui.Message(fmt.Sprintf("device node %s already exists", childDevice))
		return multistep.ActionContinue
	}

	// Get the parent's minor number using lsblk.
	out, err := exec.Command("lsblk", "-no", "MINOR", loopDevice).Output()
	if err != nil {
		ui.Error(fmt.Sprintf("failed to get parent's minor number for %s: %v", loopDevice, err))
		return multistep.ActionHalt
	}
	parentMinorStr := strings.TrimSpace(string(out))
	parentMinor, err := strconv.Atoi(parentMinorStr)
	if err != nil {
		ui.Error(fmt.Sprintf("failed to parse parent's minor number %q: %v", parentMinorStr, err))
		return multistep.ActionHalt
	}

	// Calculate the expected minor number for the partition.
	newMinor := parentMinor + partition
	ui.Message(fmt.Sprintf("creating missing device node %s with major 7 and minor %d", childDevice, newMinor))

	// Create the device node using mknod.
	out, err = exec.Command("mknod", childDevice, "b", "7", fmt.Sprintf("%d", newMinor)).CombinedOutput()
	if err != nil {
		ui.Error(fmt.Sprintf("failed to create device node %s: %v: %s", childDevice, err, string(out)))
		return multistep.ActionHalt
	}

	ui.Message(fmt.Sprintf("created missing device node %s", childDevice))
	return multistep.ActionContinue
}

// Cleanup for this step (nothing to clean up here).
func (s *StepEnsureLoopPartitionExists) Cleanup(state multistep.StateBag) {}

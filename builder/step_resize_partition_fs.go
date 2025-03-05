// step_resize_partition_fs.go
package builder

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
)

// StepResizePartitionFs expands an already partitioned image.
type StepResizePartitionFs struct {
	FromKey              string
	SelectedPartitionKey string
}

// Run the step.
func (s *StepResizePartitionFs) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packer.Ui)

	// Now loopDevice is already a device-mapper path, e.g. "/dev/mapper/loop0"
	loopDevice := state.Get(s.FromKey).(string)
	selectedPartition := state.Get(s.SelectedPartitionKey).(int)

	// Construct the partition device by appending "p<partition>"
	device := fmt.Sprintf("%sp%d", loopDevice, selectedPartition)
	ui.Message(fmt.Sprintf("Running resize2fs on partition device %s", device))

	out, err := exec.Command("resize2fs", "-f", device).CombinedOutput()
	if err != nil {
		ui.Error(fmt.Sprintf("Error while resizing partition %v: %s", err, string(out)))
		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}


// Cleanup after step execution (no cleanup needed here).
func (s *StepResizePartitionFs) Cleanup(state multistep.StateBag) {}

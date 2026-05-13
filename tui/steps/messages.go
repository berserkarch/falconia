package steps

import tea "github.com/charmbracelet/bubbletea"

// --- Navigation messages ---

// DoneMsg signals the current step is complete; advance to the next.
type DoneMsg struct{}

// BackMsg signals the user wants to go back one step.
type BackMsg struct{}

// StartInstallMsg signals the user confirmed and Phase 2 should begin.
type StartInstallMsg struct{}

// LaunchCfdiskMsg signals that cfdisk should be launched for the given disk.
type LaunchCfdiskMsg struct {
	Disk string
}

// EmitDone returns a Cmd that sends DoneMsg.
func EmitDone() tea.Cmd {
	return func() tea.Msg { return DoneMsg{} }
}

// EmitBack returns a Cmd that sends BackMsg.
func EmitBack() tea.Cmd {
	return func() tea.Msg { return BackMsg{} }
}

// EmitStartInstall returns a Cmd that sends StartInstallMsg.
func EmitStartInstall() tea.Cmd {
	return func() tea.Msg { return StartInstallMsg{} }
}

// EmitLaunchCfdisk returns a Cmd that sends LaunchCfdiskMsg.
func EmitLaunchCfdisk(disk string) tea.Cmd {
	return func() tea.Msg { return LaunchCfdiskMsg{Disk: disk} }
}

// Saveable is an optional interface for models that can persist their state
// to the shared config object. This is used when toggling advanced mode.
type Saveable interface {
	Save()
}

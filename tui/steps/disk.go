package steps

import (
	"fmt"
	"strconv"
	"strings"

	"falconia/config"
	"falconia/installer"
	"falconia/style"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type diskField int

const (
	diskFieldDisk diskField = iota
	diskFieldScheme
	diskFieldFS
	diskFieldEncrypt
	diskFieldEncPass
	diskFieldEncConf
	diskFieldSwapMode
	diskFieldSwapSize // advanced only; hidden for none/suspend modes
	diskFieldCount
)

var swapModes = []string{"none", "partition", "file", "suspend"}

// DiskModel handles disk selection, partition scheme, filesystem, and swap.
type DiskModel struct {
	cfg      *config.InstallConfig
	advanced bool

	disks   []installer.DiskInfo
	diskErr string

	cursor      diskField
	diskIdx     int // index into disks slice
	schemeIdx   int // 0=guided, 1=manual
	fsIdx       int // 0=ext4, 1=btrfs, 2=xfs
	encrypt     bool
	swapModeIdx int // index into swapModes
	swapInput   textinput.Model
	passInput   textinput.Model
	confInput   textinput.Model

	err string
}

var fsOptions = []string{"ext4", "btrfs", "xfs"}

func NewDisk(cfg *config.InstallConfig, advanced bool) DiskModel {
	ti := textinput.New()
	ti.Placeholder = "4096"
	ti.CharLimit = 8
	ti.Width = 15
	if cfg.SwapSize > 0 {
		ti.SetValue(strconv.Itoa(cfg.SwapSize))
	}

	pi := textinput.New()
	pi.Placeholder = "Encryption Password"
	pi.EchoMode = textinput.EchoPassword
	pi.EchoCharacter = '•'
	pi.Width = 30
	pi.SetValue(cfg.EncryptionPass)

	ci := textinput.New()
	ci.Placeholder = "Confirm Password"
	ci.EchoMode = textinput.EchoPassword
	ci.EchoCharacter = '•'
	ci.Width = 30
	ci.SetValue(cfg.EncryptionPass)

	disks, err := installer.ListDisks()
	diskErr := ""
	if err != nil {
		diskErr = err.Error()
	}

	// find current disk index
	diskIdx := 0
	for i, d := range disks {
		if d.Path == cfg.Disk {
			diskIdx = i
			break
		}
	}

	fsIdx := 0
	for i, f := range fsOptions {
		if f == cfg.Filesystem {
			fsIdx = i
			break
		}
	}

	schemeIdx := 0
	if cfg.PartitionScheme == "manual" {
		schemeIdx = 1
	}

	swapModeIdx := 1 // default: partition
	for i, m := range swapModes {
		if m == cfg.SwapMode {
			swapModeIdx = i
			break
		}
	}

	return DiskModel{
		cfg:         cfg,
		advanced:    advanced,
		disks:       disks,
		diskErr:     diskErr,
		diskIdx:     diskIdx,
		schemeIdx:   schemeIdx,
		fsIdx:       fsIdx,
		encrypt:     cfg.EncryptDisk,
		swapModeIdx: swapModeIdx,
		swapInput:   ti,
		passInput:   pi,
		confInput:   ci,
	}
}

func (m DiskModel) Init() tea.Cmd { return nil }

func (m DiskModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, EmitBack()

		case "up", "k", "shift+tab":
			if msg.String() == "k" && m.isTextInput() {
				break
			}
			m.cursor = m.prevField()
			m.updateFocus()
			return m, nil

		case "down", "j", "tab":
			if msg.String() == "j" && m.isTextInput() {
				break
			}
			m.cursor = m.nextField()
			m.updateFocus()
			return m, nil

		case "left", "h":
			m.cycleLeft()
		case "right", "l":
			m.cycleRight()

		case "enter":
			if m.cursor != m.lastField() {
				m.cursor = m.nextField()
				m.updateFocus()
				return m, nil
			}
			// On last field — validate and submit.
			if err := m.validate(); err != "" {
				m.err = err
				return m, nil
			}
			m.Save()
			if m.cfg.PartitionScheme == "manual" {
				return m, EmitLaunchCfdisk(m.cfg.Disk)
			}
			return m, EmitDone()
		}

		// delegate input keystrokes
		var cmd tea.Cmd
		if m.cursor == diskFieldSwapSize {
			m.swapInput, cmd = m.swapInput.Update(msg)
			return m, cmd
		}
		if m.cursor == diskFieldEncPass {
			m.passInput, cmd = m.passInput.Update(msg)
			return m, cmd
		}
		if m.cursor == diskFieldEncConf {
			m.confInput, cmd = m.confInput.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m *DiskModel) updateFocus() {
	m.swapInput.Blur()
	m.passInput.Blur()
	m.confInput.Blur()
	switch m.cursor {
	case diskFieldSwapSize:
		m.swapInput.Focus()
	case diskFieldEncPass:
		m.passInput.Focus()
	case diskFieldEncConf:
		m.confInput.Focus()
	}
}

// isTextInput reports whether the cursor is on a free-text input field.
func (m DiskModel) isTextInput() bool {
	return m.cursor == diskFieldSwapSize ||
		m.cursor == diskFieldEncPass ||
		m.cursor == diskFieldEncConf
}

// swapSizeVisible reports whether the swap size input should be shown.
func (m DiskModel) swapSizeVisible() bool {
	mode := swapModes[m.swapModeIdx]
	return m.advanced && (mode == "partition" || mode == "file")
}

// lastField returns the last navigable field given current state.
func (m DiskModel) lastField() diskField {
	if m.swapSizeVisible() {
		return diskFieldSwapSize
	}
	return diskFieldSwapMode
}

// nextField returns the field after the current one, wrapping to 0.
func (m DiskModel) nextField() diskField {
	last := m.lastField()
	if m.cursor >= last {
		return 0
	}
	next := m.cursor + 1
	// skip enc pass/conf when not encrypting
	if !m.encrypt && (next == diskFieldEncPass || next == diskFieldEncConf) {
		next = diskFieldSwapMode
	}
	// skip swapSize when not visible
	if next == diskFieldSwapSize && !m.swapSizeVisible() {
		return 0
	}
	return next
}

// prevField returns the field before the current one, wrapping to lastField.
func (m DiskModel) prevField() diskField {
	last := m.lastField()
	if m.cursor == 0 {
		return last
	}
	prev := m.cursor - 1
	// skip enc pass/conf when not encrypting
	if !m.encrypt && (prev == diskFieldEncConf || prev == diskFieldEncPass) {
		prev = diskFieldEncrypt
	}
	// skip swapSize when not visible
	if prev == diskFieldSwapSize && !m.swapSizeVisible() {
		prev = diskFieldSwapMode
	}
	return prev
}

func (m *DiskModel) cycleLeft() {
	switch m.cursor {
	case diskFieldDisk:
		if len(m.disks) > 0 {
			m.diskIdx = (m.diskIdx - 1 + len(m.disks)) % len(m.disks)
		}
	case diskFieldScheme:
		m.schemeIdx = (m.schemeIdx - 1 + 2) % 2
	case diskFieldFS:
		m.fsIdx = (m.fsIdx - 1 + len(fsOptions)) % len(fsOptions)
	case diskFieldEncrypt:
		m.encrypt = !m.encrypt
	case diskFieldSwapMode:
		m.swapModeIdx = (m.swapModeIdx - 1 + len(swapModes)) % len(swapModes)
	}
}

func (m *DiskModel) cycleRight() {
	switch m.cursor {
	case diskFieldDisk:
		if len(m.disks) > 0 {
			m.diskIdx = (m.diskIdx + 1) % len(m.disks)
		}
	case diskFieldScheme:
		m.schemeIdx = (m.schemeIdx + 1) % 2
	case diskFieldFS:
		m.fsIdx = (m.fsIdx + 1) % len(fsOptions)
	case diskFieldEncrypt:
		m.encrypt = !m.encrypt
	case diskFieldSwapMode:
		m.swapModeIdx = (m.swapModeIdx + 1) % len(swapModes)
	}
}

func (m DiskModel) validate() string {
	if len(m.disks) == 0 {
		return "No block devices found"
	}

	if m.encrypt {
		p1 := strings.TrimSpace(m.passInput.Value())
		p2 := strings.TrimSpace(m.confInput.Value())
		if p1 == "" {
			return "Encryption password cannot be empty"
		}
		if p1 != p2 {
			return "Encryption passwords do not match"
		}
	}

	if m.swapSizeVisible() {
		v := strings.TrimSpace(m.swapInput.Value())
		if v != "" {
			n, err := strconv.Atoi(v)
			if err != nil {
				return "Swap size must be a number (MiB)"
			}
			if n <= 0 {
				return "Swap size must be greater than 0 MiB"
			}
		}
	}
	return ""
}

func (m DiskModel) Save() {
	if len(m.disks) > 0 {
		d := m.disks[m.diskIdx]
		m.cfg.Disk = d.Path
		m.cfg.DiskModel = fmt.Sprintf("%s (%s)", d.Model, d.Size)
	}
	schemes := []string{"guided", "manual"}
	m.cfg.PartitionScheme = schemes[m.schemeIdx]
	m.cfg.Filesystem = fsOptions[m.fsIdx]
	// Encryption is not supported with manual partitioning: FormatDisks and
	// MountDisks skip LUKS setup in manual mode, so cfg.EncryptDisk must be
	// false to avoid building a broken bootloader config (wrong rd.luks.uuid).
	m.cfg.EncryptDisk = m.encrypt && m.cfg.PartitionScheme != "manual"
	if m.cfg.EncryptDisk {
		// Trim so the stored key matches what validate() checked; a stray leading
		// space would pass validation (trimmed != "") but corrupt the LUKS key.
		m.cfg.EncryptionPass = strings.TrimSpace(m.passInput.Value())
	} else {
		m.cfg.EncryptionPass = ""
	}

	m.cfg.SwapMode = swapModes[m.swapModeIdx]
	if m.swapSizeVisible() {
		v := strings.TrimSpace(m.swapInput.Value())
		if v == "" {
			m.cfg.SwapSize = 4096
		} else {
			n, _ := strconv.Atoi(v)
			m.cfg.SwapSize = n
		}
	}
}

func (m DiskModel) View() string {
	var b strings.Builder

	b.WriteString(style.StyleStepHeader.Render("01 — SELECT DISK") + "\n\n")

	if m.diskErr != "" {
		b.WriteString(style.StyleError.Render("  Error listing disks: "+m.diskErr) + "\n\n")
	}

	cursor := func(f diskField) string {
		if m.cursor == f {
			return style.StyleSelected.Render("▶ ")
		}
		return "  "
	}

	// --- Disk picker ---
	b.WriteString(cursor(diskFieldDisk))
	b.WriteString(style.StyleKey.Render("Disk  "))
	if len(m.disks) == 0 {
		b.WriteString(style.StyleError.Render("no devices found"))
	} else {
		d := m.disks[m.diskIdx]
		label := fmt.Sprintf("%s  %s  %s", d.Path, d.Model, d.Size)
		b.WriteString(style.StyleValue.Render(label))
		if len(m.disks) > 1 {
			b.WriteString(style.StyleMuted.Render(fmt.Sprintf("  (%d/%d)", m.diskIdx+1, len(m.disks))))
		}
	}
	b.WriteString("\n")

	// --- Scheme ---
	schemes := []string{"guided", "manual"}
	b.WriteString(cursor(diskFieldScheme))
	b.WriteString(style.StyleKey.Render("Scheme"))
	b.WriteString("  ")
	for i, s := range schemes {
		if i == m.schemeIdx {
			b.WriteString(style.ToggleOn(s))
		} else {
			b.WriteString(style.ToggleOff(s))
		}
		b.WriteString("  ")
	}
	b.WriteString("\n")

	// --- Filesystem ---
	b.WriteString(cursor(diskFieldFS))
	b.WriteString(style.StyleKey.Render("Filesys"))
	b.WriteString(" ")
	for i, f := range fsOptions {
		if i == m.fsIdx {
			b.WriteString(style.ToggleOn(f))
		} else {
			b.WriteString(style.ToggleOff(f))
		}
		b.WriteString("  ")
	}
	b.WriteString("\n")

	// --- Encryption ---
	b.WriteString(cursor(diskFieldEncrypt))
	b.WriteString(style.StyleKey.Render("Encrypt"))
	b.WriteString(" ")
	if m.encrypt {
		b.WriteString(style.ToggleOn("yes"))
		b.WriteString(style.ToggleOff("no"))
	} else {
		b.WriteString(style.ToggleOff("yes"))
		b.WriteString(style.ToggleOn("no"))
	}
	b.WriteString("\n")

	if m.encrypt {
		b.WriteString(cursor(diskFieldEncPass))
		b.WriteString(style.StyleKey.Render("Pass   "))
		b.WriteString(" ")
		b.WriteString(m.passInput.View())
		b.WriteString("\n")

		b.WriteString(cursor(diskFieldEncConf))
		b.WriteString(style.StyleKey.Render("Confirm"))
		b.WriteString(" ")
		b.WriteString(m.confInput.View())
		b.WriteString("\n")
	}

	// --- Swap mode ---
	b.WriteString(cursor(diskFieldSwapMode))
	b.WriteString(style.StyleKey.Render("Swap   "))
	b.WriteString(" ")
	for i, s := range swapModes {
		if i == m.swapModeIdx {
			b.WriteString(style.ToggleOn(s))
		} else {
			b.WriteString(style.ToggleOff(s))
		}
		b.WriteString("  ")
	}
	if swapModes[m.swapModeIdx] == "suspend" {
		b.WriteString(style.StyleMuted.Render("(auto-sized to RAM)"))
	}
	b.WriteString("\n")

	// --- Swap size (advanced, partition/file only) ---
	if m.swapSizeVisible() {
		b.WriteString(cursor(diskFieldSwapSize))
		b.WriteString(style.StyleKey.Render("Swap MiB"))
		b.WriteString(" ")
		b.WriteString(m.swapInput.View())
		b.WriteString("\n")
	}

	// --- Danger warning ---
	if len(m.disks) > 0 {
		d := m.disks[m.diskIdx]
		b.WriteString("\n")
		warning := fmt.Sprintf("⚠  ALL DATA ON %s (%s) WILL BE DESTROYED", d.Path, d.Size)
		b.WriteString(style.StyleDanger.Render(warning) + "\n")
	}

	if m.err != "" {
		b.WriteString("\n" + style.StyleError.Render("  "+m.err) + "\n")
	}

	b.WriteString("\n")
	b.WriteString(style.HelpRow(
		"↑↓/tab", "field",
		"←→", "change",
		"ctrl+a", "advanced",
		"enter", "next",
		"esc", "back",
	))

	return b.String()
}

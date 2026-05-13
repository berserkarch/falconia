package steps

import (
	"falconia/config"
	"falconia/installer"
	"falconia/style"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type partitionMappingField int

const (
	pmFieldRoot partitionMappingField = iota
	pmFieldBoot
	pmFieldSwap
	pmFieldConfirm
)

type PartitionMappingModel struct {
	cfg   *config.InstallConfig
	parts []installer.PartitionInfo
	err   string

	cursor partitionMappingField
	
	// Indices into parts slice
	rootIdx int
	bootIdx int
	swapIdx int

	// Mapping: 0=none, then 1 to len(parts)
	selections map[partitionMappingField]int
}

func NewPartitionMapping(cfg *config.InstallConfig) PartitionMappingModel {
	parts, err := installer.ListPartitions(cfg.Disk)
	
	selections := make(map[partitionMappingField]int)
	
	// Try to restore from config if possible
	findPart := func(path string) int {
		for i, p := range parts {
			if p.Path == path {
				return i + 1
			}
		}
		return 0
	}

	selections[pmFieldRoot] = findPart(cfg.MountPoints["/"])
	selections[pmFieldBoot] = findPart(cfg.MountPoints["/boot/efi"])
	// Swap is handled separately in config, but we can map it here
	
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}

	return PartitionMappingModel{
		cfg:        cfg,
		parts:      parts,
		err:        errMsg,
		selections: selections,
	}
}

func (m PartitionMappingModel) Init() tea.Cmd { return nil }

func (m PartitionMappingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, EmitBack()
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			} else {
				m.cursor = pmFieldConfirm
			}
		case "down", "j", "tab":
			if m.cursor < pmFieldConfirm {
				m.cursor++
			} else {
				m.cursor = 0
			}
		case "left", "h":
			if m.cursor < pmFieldConfirm {
				m.selections[m.cursor] = (m.selections[m.cursor] - 1 + len(m.parts) + 1) % (len(m.parts) + 1)
			}
		case "right", "l":
			if m.cursor < pmFieldConfirm {
				m.selections[m.cursor] = (m.selections[m.cursor] + 1) % (len(m.parts) + 1)
			}
		case "enter":
			if m.cursor == pmFieldConfirm {
				if m.selections[pmFieldRoot] == 0 {
					m.err = "Root (/) partition is required"
					return m, nil
				}
				if m.cfg.Firmware == "uefi" && m.selections[pmFieldBoot] == 0 {
					m.err = "EFI Boot partition (/boot/efi) is required for UEFI"
					return m, nil
				}
				m.Save()
				return m, EmitDone()
			}
			m.cursor = pmFieldConfirm
		}
	}
	return m, nil
}

func (m PartitionMappingModel) Save() {
	m.cfg.MountPoints = make(map[string]string)
	
	getPart := func(field partitionMappingField) string {
		idx := m.selections[field]
		if idx > 0 && idx <= len(m.parts) {
			return m.parts[idx-1].Path
		}
		return ""
	}

	if p := getPart(pmFieldRoot); p != "" {
		m.cfg.MountPoints["/"] = p
	}
	if p := getPart(pmFieldBoot); p != "" {
		m.cfg.MountPoints["/boot/efi"] = p
	}
	if p := getPart(pmFieldSwap); p != "" {
		m.cfg.MountPoints["swap"] = p
	}
}

func (m PartitionMappingModel) View() string {
	var b strings.Builder
	b.WriteString(style.StyleStepHeader.Render("01b — MAP PARTITIONS") + "\n\n")

	if m.err != "" && m.cursor != pmFieldConfirm {
		b.WriteString(style.StyleError.Render("  "+m.err) + "\n\n")
	}

	renderField := func(field partitionMappingField, label string) {
		cursor := "  "
		if m.cursor == field {
			cursor = style.StyleSelected.Render("▶ ")
		}
		b.WriteString(cursor + style.StyleKey.Render(fmt.Sprintf("%-10s", label)))
		
		idx := m.selections[field]
		if idx == 0 {
			b.WriteString(style.StyleMuted.Render("[ none ]"))
		} else {
			p := m.parts[idx-1]
			b.WriteString(style.StyleSelected.Render(fmt.Sprintf("[%s (%s)]", p.Path, p.Size)))
		}
		b.WriteString("\n")
	}

	renderField(pmFieldRoot, "Root (/)")
	if m.cfg.Firmware == "uefi" {
		renderField(pmFieldBoot, "Boot EFI")
	}
	renderField(pmFieldSwap, "Swap")

	b.WriteString("\n")
	
	confirmCursor := "  "
	if m.cursor == pmFieldConfirm {
		confirmCursor = style.StyleSelected.Render("▶ ")
	}
	b.WriteString(confirmCursor + style.StyleButtonActive.Render(" Confirm Mapping ") + "\n")

	if m.err != "" && m.cursor == pmFieldConfirm {
		b.WriteString("\n" + style.StyleError.Render("  "+m.err) + "\n")
	}

	b.WriteString("\n")
	b.WriteString(style.HelpRow("↑↓", "field", "←→", "pick partition", "enter", "next", "esc", "back"))

	return b.String()
}

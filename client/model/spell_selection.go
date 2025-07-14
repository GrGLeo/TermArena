package model

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	upKey    = key.NewBinding(key.WithKeys("up", "k"))
	downKey  = key.NewBinding(key.WithKeys("down", "j"))
	enterKey = key.NewBinding(key.WithKeys("enter"))
)

// SpellSelectionModel manages the state of the spell selection UI.
type SpellSelectionModel struct {
	styles          *Styles
	Spells          []Spell
	FocusedIndex    int
	SelectedIndices [2]int
	ActiveSelection int // 0 or 1, for the next slot to fill
	height, width   int
}

func (m *SpellSelectionModel) SetDimension(height, width int) {
	m.height = height
	m.width = width
}

func NewSpellSelection(styles *Styles) SpellSelectionModel {
	return SpellSelectionModel{
		styles:          styles,
		Spells:          availableSpells,
		FocusedIndex:    0,
		SelectedIndices: [2]int{-1, -1},
		ActiveSelection: 0,
	}
}

func (m SpellSelectionModel) Init() tea.Cmd {
	return nil
}

type SpellsSelectedMsg struct {
	SpellIDs [2]int
}

func (m SpellSelectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, upKey):
			if m.FocusedIndex > 0 {
				m.FocusedIndex--
			}
		case key.Matches(msg, downKey):
			if m.FocusedIndex < len(m.Spells)-1 {
				m.FocusedIndex++
			}
		case key.Matches(msg, enterKey):
			if m.SelectedIndices[0] != m.FocusedIndex && m.SelectedIndices[1] != m.FocusedIndex {
				m.SelectedIndices[m.ActiveSelection] = m.FocusedIndex
				m.ActiveSelection = (m.ActiveSelection + 1) % 2
			}

			// If both spells are selected, send a message
			if m.SelectedIndices[0] != -1 && m.SelectedIndices[1] != -1 {
				spellIDs := [2]int{
					m.Spells[m.SelectedIndices[0]].ID,
					m.Spells[m.SelectedIndices[1]].ID,
				}
				return m, func() tea.Msg {
					return SpellsSelectedMsg{SpellIDs: spellIDs}
				}
			}
		}
	}
	return m, nil
}

func (m SpellSelectionModel) View() string {
	var left, right strings.Builder

	// Left Panel: List of available spells
	left.WriteString("Choose Your Spells\n\n")
	for i, spell := range m.Spells {
		selectedChar := lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Render("> ")
		cursor := "  "
		if m.FocusedIndex == i {
			cursor = selectedChar
		}

		selected := " "
		if m.SelectedIndices[0] == i || m.SelectedIndices[1] == i {
			selected = "X"
		}

		spellNameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		if m.FocusedIndex == i {
			spellNameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
		}

		left.WriteString(fmt.Sprintf("%s [%s] %s\n", cursor, selected, spellNameStyle.Render(spell.Name)))
	}

	// Right Panel: Details of the focused spell
	if m.FocusedIndex >= 0 && m.FocusedIndex < len(m.Spells) {
		right.WriteString(m.Spells[m.FocusedIndex].String())
	}

	// Calculate content height, assuming tabs take up 2 lines
	optionsStyle := lipgloss.NewStyle().
		Align(lipgloss.Left).
		Padding(1, 0)

	instructionsStyle := lipgloss.NewStyle().
		Align(lipgloss.Left).
		Border(lipgloss.NormalBorder(), true, true, true, true).
		BorderForeground(m.styles.BorderColor).
		Padding(1, 0)

	layout := lipgloss.JoinHorizontal(
		lipgloss.Center,
		optionsStyle.Render(left.String()),
		instructionsStyle.Render(right.String()),
	)

	return layout
}

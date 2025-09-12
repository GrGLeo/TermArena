package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Player struct {
	username           string
	spellOne, spellTwo string
}

func NewPlayer(username string) *Player {
	return &Player{
		username: username,
	}
}

func (p *Player) UpdateSpell(spellOne, spellTwo string) {
	p.spellOne = spellOne
	p.spellTwo = spellTwo
}

func (p *Player) Display() string {
	return fmt.Sprintf("%s\n- %s\n- %s", p.username, p.spellOne, p.spellTwo)
}

type TeamPanel struct {
	team      string
	players   []*Player
	maxPlayer int
}

func NewTeamPanel(team string, maxPlayer int) *TeamPanel {
	return &TeamPanel{
		team:      team,
		maxPlayer: maxPlayer,
	}
}

func (tp *TeamPanel) AddPlayer(player *Player) {
	tp.players = append(tp.players, player)
}

func (tp *TeamPanel) Display() string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Team: %s (%d/%d players)\n", tp.team, len(tp.players), tp.maxPlayer))

	// Display all 4 slots
	for i := 0; i < tp.maxPlayer; i++ {
		if i < len(tp.players) {
			// Show actual player (takes 3 lines)
			builder.WriteString(fmt.Sprintf("  %d. %s\n", i+1, tp.players[i].Display()))
		} else {
			// Show placeholder for empty slot (3 lines to match player display)
			builder.WriteString(fmt.Sprintf("  %d. [Empty Slot]\n", i+1))
			builder.WriteString("     -\n")
			builder.WriteString("     -\n")
		}
	}

	return builder.String()
}

// Spell represents the data for a single champion ability.
type Spell struct {
	ID          int
	Name        string
	Description string
}

// String returns a formatted string for the spell's stats.
func (s Spell) String() string {
	return fmt.Sprintf(
		"Name: %s\n%s",
		s.Name,
		s.Description,
	)
}

// GetFormattedDescription returns a consistently formatted description for display
func (s Spell) GetFormattedDescription() string {
	lines := strings.Split(s.Description, "\n")
	var formatted strings.Builder

	formatted.WriteString(fmt.Sprintf("Name: %s\n", s.Name))
	for _, line := range lines {
		formatted.WriteString(line + "\n")
	}

	return formatted.String()
}

// availableSpells holds the hardcoded data for all spells in the game.
var availableSpells = []Spell{
	{
		ID:          0,
		Name:        "Freeze Wall",
		Description: "Mana Cost: 50\nCooldown: 10s\nDamage: 20 (+80% Ratio)\nEffect: Stun 1s",
	},
	{
		ID:          1,
		Name:        "Fireball",
		Description: "Mana Cost: 30\nCooldown: 5s\nDamage: 40 (+60% Ratio)\nEffect: Direct damage",
	},
	{
		ID:          2,
		Name:        "Healing Wave",
		Description: "Mana Cost: 40\nCooldown: 15s\nHeal: 50 (+30% Ratio)\nEffect: Area healing",
	},
	{
		ID:          3,
		Name:        "Whirlwind",
		Description: "Mana Cost: 10\nCooldown: 10s\nDamage: 10 (+30% Ratio)\nEffect: Multi-target",
	},
	{
		ID:          4,
		Name:        "Pierce",
		Description: "Mana Cost: 5\nCooldown: 10s\nDamage: 10% Ratio/s\nEffect: DoT 5s",
	},
}

// SpellSelectionModel manages the state of the spell selection UI.
type SpellSelectionModel struct {
	Spells          []Spell
	FocusedIndex    int
	SelectedIndices [2]int
	ActiveSelection int // 0 or 1, for the next slot to fill
}

func NewSpellSelectionModel() SpellSelectionModel {
	return SpellSelectionModel{
		Spells:          availableSpells,
		FocusedIndex:    0,
		SelectedIndices: [2]int{-1, -1},
		ActiveSelection: 0,
	}
}

func (m SpellSelectionModel) View() string {
	// Left Panel: Spell selection list
	var leftBuilder strings.Builder
	leftBuilder.WriteString("Spell Selection\n\n")

	// Show selected spells
	leftBuilder.WriteString("Selected Spells:\n")
	for i := 0; i < 2; i++ {
		if m.SelectedIndices[i] != -1 {
			spell := m.Spells[m.SelectedIndices[i]]
			leftBuilder.WriteString(fmt.Sprintf("  %d. %s\n", i+1, spell.Name))
		} else {
			leftBuilder.WriteString(fmt.Sprintf("  %d. [Not Selected]\n", i+1))
		}
	}
	leftBuilder.WriteString("\n")

	// Show available spells
	leftBuilder.WriteString("Available Spells:\n")
	for i, spell := range m.Spells {
		cursor := "  "
		if m.FocusedIndex == i {
			cursor = "> "
		}

		selected := " "
		if m.SelectedIndices[0] == i || m.SelectedIndices[1] == i {
			selected = "X"
		}

		spellStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		if m.FocusedIndex == i {
			spellStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
		}

		leftBuilder.WriteString(fmt.Sprintf("%s[%s] %s\n", cursor, selected, spellStyle.Render(spell.Name)))
	}

	// Right Panel: Spell description
	var rightBuilder strings.Builder
	rightBuilder.WriteString("Spell Details\n\n")

	if m.FocusedIndex >= 0 && m.FocusedIndex < len(m.Spells) {
		rightBuilder.WriteString(m.Spells[m.FocusedIndex].GetFormattedDescription())
	} else {
		rightBuilder.WriteString("Select a spell to view details")
	}

	// Create styles for the two panels
	leftStyle := lipgloss.NewStyle().
		Width(35).
		Align(lipgloss.Left).
		Padding(1, 2)

	rightStyle := lipgloss.NewStyle().
		Width(35).
		Align(lipgloss.Left).
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2)

	// Combine left and right panels horizontally
	layout := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftStyle.Render(leftBuilder.String()),
		rightStyle.Render(rightBuilder.String()),
	)

	return layout
}

func (m *SpellSelectionModel) Update(msg tea.Msg) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.FocusedIndex > 0 {
				m.FocusedIndex--
			}
		case "down", "j":
			if m.FocusedIndex < len(m.Spells)-1 {
				m.FocusedIndex++
			}
		case "enter":
			if m.SelectedIndices[0] != m.FocusedIndex && m.SelectedIndices[1] != m.FocusedIndex {
				m.SelectedIndices[m.ActiveSelection] = m.FocusedIndex
				m.ActiveSelection = (m.ActiveSelection + 1) % 2
			}
		}
	}
}

type model struct {
	cursor         int
	blueTeam       *TeamPanel
	redTeam        *TeamPanel
	activeTeam     int // 0 = blue, 1 = red
	spellSelection SpellSelectionModel
	activePanel    int // 0 = blue, 1 = red, 2 = spells
	width, height  int // Terminal dimensions
	viewport       viewport.Model
	textInput      textinput.Model
}

func initialModel() model {
	// Initialize viewport for messaging
	vp := viewport.New(76, 8) // Fixed size: 80 width - 4 for padding
	vp.SetContent("Welcome to TermArena!\n\nThis is the messaging area.\nMessages will appear here...\n")

	// Initialize text input for chat
	ti := textinput.New()
	ti.Placeholder = "Type your message here..."
	ti.Width = 76 // Fixed size: 80 width - 4 for padding
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return model{
		blueTeam:       NewTeamPanel("Blue", 4),
		redTeam:        NewTeamPanel("Red", 4),
		activeTeam:     0, // Start with Blue team
		spellSelection: NewSpellSelectionModel(),
		activePanel:    0,   // Start with Blue team panel
		width:          120, // Default width
		height:         30,  // Default height
		viewport:       vp,
		textInput:      ti,
	}
}

func (m model) Init() tea.Cmd {
	return tea.ClearScreen
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Update terminal dimensions
		m.width = msg.Width
		m.height = msg.Height
		// Note: Messaging components use fixed sizes, not dynamic
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			// Switch between panels: Blue -> Spells -> Red -> Blue
			m.activePanel = (m.activePanel + 1) % 3
			if m.activePanel == 0 {
				m.activeTeam = 0 // Blue team
			} else if m.activePanel == 1 {
				// Spells panel
			} else if m.activePanel == 2 {
				m.activeTeam = 1 // Red team
			}
		case "a":
			// Add a sample player to the active team (only when team panels are active)
			if m.activePanel == 0 || m.activePanel == 2 {
				player := NewPlayer(fmt.Sprintf("Player%d", len(m.blueTeam.players)+len(m.redTeam.players)+1))
				player.UpdateSpell("Fireball", "Heal")
				if m.activePanel == 0 {
					m.blueTeam.AddPlayer(player)
				} else {
					m.redTeam.AddPlayer(player)
				}
			}
		default:
			// Handle spell selection when spells panel is active
			if m.activePanel == 1 {
				m.spellSelection.Update(msg)
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	// Create pink rounded border style
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")). // Pink color
		Padding(1, 2).
		Margin(0, 1)

	// Create active indicator style
	activeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")). // Pink color
		Bold(true)

	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")) // Gray color

	// Blue team panel
	blueTitle := "Blue Team"
	if m.activePanel == 0 {
		blueTitle = activeStyle.Render("> Blue Team")
	} else {
		blueTitle = normalStyle.Render("  Blue Team")
	}

	blueContent := blueTitle + "\n" + m.blueTeam.Display()
	bluePanel := borderStyle.Render(blueContent)

	// Spell selection panel
	spellTitle := "Spell Selection"
	if m.activePanel == 1 {
		spellTitle = activeStyle.Render("> Spell Selection")
	} else {
		spellTitle = normalStyle.Render("  Spell Selection")
	}

	spellContent := spellTitle + "\n" + m.spellSelection.View()
	spellPanel := borderStyle.Render(spellContent)

	// Red team panel
	redTitle := "Red Team"
	if m.activePanel == 2 {
		redTitle = activeStyle.Render("> Red Team")
	} else {
		redTitle = normalStyle.Render("  Red Team")
	}

	redContent := redTitle + "\n" + m.redTeam.Display()
	redPanel := borderStyle.Render(redContent)

	// Combine all three panels side by side
	allPanelsView := lipgloss.JoinHorizontal(lipgloss.Top, bluePanel, spellPanel, redPanel)

	// Messaging section
	messagingStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Margin(1, 0).
		Width(138) // Fixed width for messaging

	messagingTitle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true).
		Render("💬 Chat")

	messagingContent := messagingTitle + "\n" + m.viewport.View()

	// Text input section
	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Margin(0, 0, 1, 0).
		Width(138) // Fixed width for text input

	inputContent := inputStyle.Render(m.textInput.View())

	// Combine everything vertically
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		allPanelsView,
		messagingStyle.Render(messagingContent),
		inputContent,
	)

	// Center the entire content on screen using lipgloss.Place
	// Using actual terminal dimensions for proper centering
	centeredContent := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Top, content)

	return centeredContent
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

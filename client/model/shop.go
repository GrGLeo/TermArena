package model

import (
	"fmt"
	"strings"

	"github.com/GrGLeo/ctf/client/communication"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	shopUpKey    = key.NewBinding(key.WithKeys("up", "k"))
	shopDownKey  = key.NewBinding(key.WithKeys("down", "j"))
	shopEnterKey = key.NewBinding(key.WithKeys("enter"))
	shopBackKey  = key.NewBinding(key.WithKeys("esc", "p"))
)

// ShopModel manages the state of the shop UI.
type ShopModel struct {
	styles               *Styles
	Items                []Item
	FocusedIndex         int
	height, width        int
	health, mana         int
	attack_damage, armor int
	gold                 int
}

func (m *ShopModel) SetDimension(height, width int) {
	m.height = height
	m.width = width
}

func NewShopModel(styles *Styles, health, mana, attack_damage, armor, gold int) ShopModel {
	return ShopModel{
		styles:        styles,
		Items:         availableItems,
		FocusedIndex:  0,
		health:        health,
		mana:          mana,
		attack_damage: attack_damage,
		armor:         armor,
		gold:          gold,
	}
}

func (m ShopModel) Init() tea.Cmd {
	return nil
}

type ItemPurchasedMsg struct {
	ItemID int
}

func (m ShopModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, shopUpKey):
			if m.FocusedIndex > 0 {
				m.FocusedIndex--
			}
		case key.Matches(msg, shopDownKey):
			if m.FocusedIndex < len(m.Items)-1 {
				m.FocusedIndex++
			}
		case key.Matches(msg, shopEnterKey):
			if m.FocusedIndex >= 0 && m.FocusedIndex < len(m.Items) {
				selectedItem := m.Items[m.FocusedIndex]
				fmt.Printf("Attempting to purchase: %s for %d gold ", selectedItem.Name, selectedItem.Cost)
				// TODO: send a request to purchase item
				// communication.SendPurchaseItemPacket(m.conn, selectedItem.ID)
				return m, func() tea.Msg {
					return ItemPurchasedMsg{ItemID: selectedItem.ID}
				}
			}
		case key.Matches(msg, shopBackKey):
			return m, func() tea.Msg {
				return communication.BackToGameMsg{}
			}
		}
	}
	return m, nil
}

func (m ShopModel) View() string {
	var left, right, bottom strings.Builder

	// Left Panel: List of available items
	left.WriteString("Shop - Available Items \n")
	for i, item := range m.Items {
		selectedChar := lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Render("> ")
		cursor := "  "
		if m.FocusedIndex == i {
			cursor = selectedChar
		}

		itemNameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		if m.FocusedIndex == i {
			itemNameStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
		}

		left.WriteString(fmt.Sprintf("%s %s \n", cursor, itemNameStyle.Render(item.Name)))
	}

	// Right Panel: Details of the focused item
	if m.FocusedIndex >= 0 && m.FocusedIndex < len(m.Items) {
		right.WriteString(m.Items[m.FocusedIndex].String())
	}

	// Bottom Panel: Player stats
	bottom.WriteString(fmt.Sprintf("Health: %d | Mana: %d | Attack damage: %d | Armor: %d", m.health, m.mana, m.attack_damage, m.armor))
  bottom.WriteString(fmt.Sprintf("\nGold: %d", m.gold))

	optionsStyle := lipgloss.NewStyle().
		Align(lipgloss.Left).
		Padding(1, 0)

	detailsStyle := lipgloss.NewStyle().
		Align(lipgloss.Left).
		Border(lipgloss.NormalBorder(), true, true, true, true).
		BorderForeground(m.styles.BorderColor).
		Padding(1, 0)

	statsStyle := lipgloss.NewStyle().
		Align(lipgloss.Left).
		Border(lipgloss.NormalBorder(), true, false, false).
		BorderForeground(m.styles.BorderColor).
		Padding(1, 0)

	baseLayout := lipgloss.JoinHorizontal(
		lipgloss.Center,
		optionsStyle.Render(left.String()),
		detailsStyle.Render(right.String()),
	)

	layout := lipgloss.JoinVertical(
		lipgloss.Center,
		baseLayout,
		statsStyle.Render(bottom.String()),
	)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		layout,
	)
}

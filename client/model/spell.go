package model

import "fmt"

// Spell represents the data for a single champion ability.
type Spell struct {
	ID          int
	Name        string
	Description string
}

func (s Spell) String() string {
	return fmt.Sprintf(
		"Name: %s\n%s",
		s.Name,
		s.Description,
	)
}

// availableSpells holds the hardcoded data for all spells in the game.
var availableSpells = []Spell{
	{
		ID:          0,
		Name:        "Freeze Wall",
		Description: "Mana Cost: 50\nCooldown: 10s\nDamage: 20 (+80% Ratio)\nStun: 1 second",
	},
	{
		ID:          1,
		Name:        "Fireball",
		Description: "Mana Cost: 30\nCooldown: 5s \nDamage: 40 (+60% Ratio)\n ",
	},
	{
		ID:          2,
		Name:        "Healing Wave",
		Description: "Mana Cost: 40\nCooldown: 15s\nHeal: 50 (+30% Ratio)  \n ",
	},
	{
		ID:          3,
		Name:        "Whirlwind",
		Description: "Mana Cost: 10\nCooldown: 10s\nDamage: 10 (+30% Ratio)\n ",
	},
	{
		ID:          4,
		Name:        "Pierce",
		Description: "Mana Cost: 5\nCooldown: 10s\nDoT: 10% Ratio         \nEffect: 5 second",
	},
	{
		ID:          5,
		Name:        "Electric Pulse",
		Description: "Mana Cost: 15\nCooldown: 8s\nStun: 0.5 seconds      \nCast Time: 0.5s",
	},
}

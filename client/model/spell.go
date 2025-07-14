package model

import "fmt"

// Spell represents the data for a single champion ability.
type Spell struct {
	ID           int
	Name         string
	ManaCost     int
	CooldownSecs int
	Range        int
	Width        int
	Speed        int
	BaseDamage   int
	DamageRatio  float32
	StunDuration int
}

// String returns a formatted string for the spell's stats.
func (s Spell) String() string {
	return fmt.Sprintf(
		"Name: %s\nMana Cost: %d\nCooldown: %ds\nRange: %d\nWidth: %d\nSpeed: %d\nBase Damage: %d\nDamage Ratio: %.2f\nStun: %d seconds",
		s.Name,
		s.ManaCost,
		s.CooldownSecs,
		s.Range,
		s.Width,
		s.Speed,
		s.BaseDamage,
		s.DamageRatio,
		s.StunDuration,
	)
}

// availableSpells holds the hardcoded data for all spells in the game.
// This will be used to populate the selection UI.
var availableSpells = []Spell{
	{
		ID:           0,
		Name:         "Freeze Wall",
		ManaCost:     50,
		CooldownSecs: 10,
		Range:        10,
		Width:        5,
		Speed:        1,
		BaseDamage:   20,
		DamageRatio:  0.8,
		StunDuration: 1,
	},
	{
		ID:           1,
		Name:         "Fireball",
		ManaCost:     30,
		CooldownSecs: 5,
		Range:        15,
		Width:        1,
		Speed:        3,
		BaseDamage:   40,
		DamageRatio:  0.6,
		StunDuration: 0, // No stun
	},
}

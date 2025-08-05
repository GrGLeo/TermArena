package model

import "fmt"

// Item represents the data for a single item available in the shop.
type Item struct {
	ID          int
	Name        string
	Description string
	Cost        int
	Damage      int
	Armor       int
	Health      int
}

func (i Item) String() string {
	s := fmt.Sprintf(
		"Name: %s\nCost: %d\nDescription: %s",
		i.Name,
		i.Cost,
		i.Description,
	)
	if i.Damage > 0 {
		s += fmt.Sprintf("\nDamage: %d", i.Damage)
	}
	if i.Armor > 0 {
		s += fmt.Sprintf("\nArmor: %d", i.Armor)
	}
	if i.Health > 0 {
		s += fmt.Sprintf("\nHealth: %d", i.Health)
	}
	return s
}

var availableItems = []Item{
	{
		ID:          0,
		Name:        "Sword of Power",
		Description: "Increases attack damage.          ",
		Cost:        200,
		Damage:      10,
	},
	{
		ID:          1,
		Name:        "Armor of Resilience",
		Description: "Increases defense.                ",
		Cost:        200,
		Armor:       10,
	},
	{
		ID:          2,
		Name:        "Shield of Valor",
		Description: "Increases both armor and health",
		Cost:        200,
		Armor:       5,
		Health:      50,
	},
}

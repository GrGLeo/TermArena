package model

import "fmt"

// Item represents the data for a single item available in the shop.
type Item struct {
	ID          int
	Name        string
	Description string
}

func (i Item) String() string {
	return fmt.Sprintf(
		"Name: %s\n%s",
		i.Name,
		i.Description,
	)
}

var availableItems = []Item{
	{
		ID:   1,
		Name: "Sword of Power",
		Description: "Cost: 300\nDamage: 10",
	},
	{
		ID:   2,
		Name: "Armor of Resilience",
		Description: "Cost: 200\nArmor: 10",
	},
	{
		ID:   3,
		Name: "Health Stone",
		Description: "Cost: 400\nHealth: 50",
	},
	{
		ID:   4,
		Name: "Mana Pendant",
		Description: "Cost: 400\nMana: 25",
	},
	{
		ID:   5,
		Name: "Dagger",
		Description: "Cost: 200\nAttack Speed: 200",
	},
	{
		ID:   6,
		Name: "Vial of Renewal",
		Description: "Cost: 150\nHealth regeneration increase: 100%",
	},
	{
		ID:   7,
		Name: "Shield of Valor",
		Description: "Cost: 200\nArmor: 5\nHealth: 50",
	},
  {
    ID: 8,
    Name: "Double-edged Sword",
    Description: "Cost: 700\nUpgrade Cost: 100\nDamage:25\nRequired: 2x'Simple Sword'",
  },
}

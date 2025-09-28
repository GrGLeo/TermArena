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
		ID:          1,
		Name:        "Simple Sword",
		Description: "Cost: 350\nDamage: 10",
	},
	{
		ID:          2,
		Name:        "Tome of Power",
		Description: "Cost: 450\nMagic Power: 10",
	},
	{
		ID:          3,
		Name:        "Leather Armor",
		Description: "Cost: 300\nArmor: 10",
	},
	{
		ID:          4,
		Name:        "Health Stone",
		Description: "Cost: 400\nHealth: 50",
	},
	{
		ID:          5,
		Name:        "Mana Pendant",
		Description: "Cost: 400\nMana: 25",
	},
	{
		ID:          6,
		Name:        "Dagger",
		Description: "Cost: 200\nAttack Speed: 200",
	},
	{
		ID:          7,
		Name:        "Vial of Renewal",
		Description: "Cost: 150\nHealth regeneration increase: 100%",
	},
	{
		ID:          8,
		Name:        "Shield of Valor",
		Description: "Cost: 800\nUpgrade Cost: 200\nHealth: 80\nArmor: 15\nRequired: 'Leather Armor | Health Stone'",
	},
	{
		ID:          9,
		Name:        "Double-edged Sword",
		Description: "Cost: 800\nUpgrade Cost: 100\nDamage: 25\nRequired: 2x'Simple Sword'",
	},
}

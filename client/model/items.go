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
		Description: "Description: Increases attack damage.\nCost: 200\nDamage: 10",
	},
	{
		ID:   2,
		Name: "Armor of Resilience",
		Description: "Description: Increases defense.\nCost: 200\nArmor: 10",
	},
	{
		ID:   3,
		Name: "Health Stone",
		Description: "Description: Increases health.\nCost: 400\nHealth: 50",
	},
	{
		ID:   4,
		Name: "Mana Pendant",
		Description: "Description: Increases mana.\nCost: 400\nMana: 25",
	},
	{
		ID:   5,
		Name: "Dagger",
		Description: "Description: Increases attack speed.\nCost: 200\nAttack Speed: 200",
	},
	{
		ID:   6,
		Name: "Vial of Renewal",
		Description: "Description: Increases health regeneration.\nCost: 150\nHealth regeneration increase: 100%",
	},
	{
		ID:   7,
		Name: "Shield of Valor",
		Description: "Description: Increases armor and health.\nCost: 200\nArmor: 5\nHealth: 50",
	},
}

use strum_macros::EnumIter;

pub type PlayerId = usize;
pub type MinionId = usize;
pub type FlagId = usize;
pub type TowerId = usize;

#[derive(Debug, Clone, Copy, PartialEq, Eq, EnumIter)]
pub enum Team {
    Blue,
    Red,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum BaseTerrain {
    Wall,
    Floor,
    Bush,
    TowerDestroyed,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum CellContent {
    Champion(PlayerId, Team),
    Minion(MinionId, Team),
    Flag(FlagId, Team),
    Tower(TowerId, Team),
    Base(Team),
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum CellAnimation {
    MeleeHit,
    TowerHit,
}

#[derive(Debug, Clone)]
pub struct Cell {
    pub position: (u16, u16),
    pub base: BaseTerrain,
    pub content: Option<CellContent>,
    pub animation: Option<CellAnimation>,
}

impl Cell {
    pub fn new(base: BaseTerrain, position: (u16, u16)) -> Self {
        Cell {
            position,
            base,
            content: None,
            animation: None,
        }
    }

    pub fn is_passable(&self) -> bool {
        match self.base {
            BaseTerrain::Wall => false,
            BaseTerrain::TowerDestroyed => false,
            BaseTerrain::Floor => self.content.is_none(),
            BaseTerrain::Bush => self.content.is_none(),
        }
    }
}

#[derive(Debug, Copy, Clone, PartialEq, Eq)]
#[repr(u8)]
pub enum EncodedCellValue {
    Wall = 0,
    Floor = 1,
    Bush = 2,
    TowerDestroyed = 3,
    Champion = 4,
    Flag = 7,
    Tower = 8,
    MeleeHitAnimation = 9,
    TowerHitAnimation = 10,
    BaseBlue = 11,
    BaseRed = 12,
    // Minion health values (100-115)
    MinionBlueHealth1 = 100,
    MinionBlueHealth2 = 101,
    MinionBlueHealth3 = 102,
    MinionBlueHealth4 = 103,
    MinionBlueHealth5 = 104,
    MinionBlueHealth6 = 105,
    MinionBlueHealth7 = 106,
    MinionBlueHealth8 = 107,
    MinionRedHealth1 = 108,
    MinionRedHealth2 = 109,
    MinionRedHealth3 = 110,
    MinionRedHealth4 = 111,
    MinionRedHealth5 = 112,
    MinionRedHealth6 = 113,
    MinionRedHealth7 = 114,
    MinionRedHealth8 = 115,
    MinionPlaceholder = 255, // Temporary placeholder
}

impl EncodedCellValue {
    pub fn from_health_level(level: u8, team: Team) -> Self {
        match team {
            Team::Blue => match level {
                1 => EncodedCellValue::MinionBlueHealth1,
                2 => EncodedCellValue::MinionBlueHealth2,
                3 => EncodedCellValue::MinionBlueHealth3,
                4 => EncodedCellValue::MinionBlueHealth4,
                5 => EncodedCellValue::MinionBlueHealth5,
                6 => EncodedCellValue::MinionBlueHealth6,
                7 => EncodedCellValue::MinionBlueHealth7,
                8 => EncodedCellValue::MinionBlueHealth8,
                _ => EncodedCellValue::MinionPlaceholder, // Should not happen with proper health_level calculation
            },
            Team::Red => match level {
                1 => EncodedCellValue::MinionRedHealth1,
                2 => EncodedCellValue::MinionRedHealth2,
                3 => EncodedCellValue::MinionRedHealth3,
                4 => EncodedCellValue::MinionRedHealth4,
                5 => EncodedCellValue::MinionRedHealth5,
                6 => EncodedCellValue::MinionRedHealth6,
                7 => EncodedCellValue::MinionRedHealth7,
                8 => EncodedCellValue::MinionRedHealth8,
                _ => EncodedCellValue::MinionPlaceholder, // Should not happen
            },
        }
    }
}

impl From<&Cell> for EncodedCellValue {
    fn from(cell: &Cell) -> Self {
        if let Some(animation) = &cell.animation {
            match animation {
                CellAnimation::MeleeHit => EncodedCellValue::MeleeHitAnimation,
                CellAnimation::TowerHit => EncodedCellValue::TowerHitAnimation,
            }
        } else if let Some(content) = &cell.content {
            match content {
                CellContent::Champion(_, _) => EncodedCellValue::Champion,
                CellContent::Minion(_, _) => EncodedCellValue::MinionPlaceholder, // Placeholder for now
                CellContent::Flag(_, _) => EncodedCellValue::Flag,
                CellContent::Tower(_, _) => EncodedCellValue::Tower,
                CellContent::Base(team) => match team {
                    Team::Blue => EncodedCellValue::BaseBlue,
                    Team::Red => EncodedCellValue::BaseRed,
                },
            }
        } else {
            match cell.base {
                BaseTerrain::Wall => EncodedCellValue::Wall,
                BaseTerrain::Floor => EncodedCellValue::Floor,
                BaseTerrain::Bush => EncodedCellValue::Bush,
                BaseTerrain::TowerDestroyed => EncodedCellValue::TowerDestroyed,
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::game::cell::Team; // Assuming Team enum is in cell.rs

    #[test]
    fn test_new_cell() {
        let base = BaseTerrain::Floor;
        let position = (10, 20); // Provide a dummy position for the test
        let cell = Cell::new(base, position); // Updated call

        assert_eq!(cell.position, position); // Assert the new position field
        assert_eq!(cell.base, base);
        assert!(cell.content.is_none());
        assert!(cell.animation.is_none());
    }

    #[test]
    fn test_is_passable() {
        let dummy_position = (0, 0); // Dummy position for these tests

        // Floor with no content should be passable
        let floor_cell = Cell::new(BaseTerrain::Floor, dummy_position); // Updated call
        assert!(
            floor_cell.is_passable(),
            "Floor with no content should be passable"
        );

        // Floor with content should not be passable
        let floor_with_content = Cell {
            position: dummy_position, // Added position
            base: BaseTerrain::Floor,
            content: Some(CellContent::Champion(1, Team::Red)),
            animation: None,
        };
        assert!(
            !floor_with_content.is_passable(),
            "Floor with content should not be passable"
        );

        // Wall should not be passable (regardless of content)
        let wall_cell = Cell::new(BaseTerrain::Wall, dummy_position); // Updated call
        assert!(!wall_cell.is_passable(), "Wall should not be passable");

        let wall_with_content = Cell {
            position: dummy_position, // Added position
            base: BaseTerrain::Wall,
            content: Some(CellContent::Champion(1, Team::Red)),
            animation: None,
        };
        assert!(
            !wall_with_content.is_passable(),
            "Wall with content should not be passable"
        );

        // Bush with no content should be passable
        let bush_cell = Cell::new(BaseTerrain::Bush, dummy_position); // Updated call
        assert!(
            bush_cell.is_passable(),
            "Bush with no content should be passable"
        );

        // Bush with content should not be passable
        let bush_with_content = Cell {
            position: dummy_position, // Added position
            base: BaseTerrain::Bush,
            content: Some(CellContent::Champion(1, Team::Red)),
            animation: None,
        };
        assert!(
            !bush_with_content.is_passable(),
            "Bush with content should not be passable"
        );

        // TowerDestroyed should not be passable (regardless of content)
        let tower_destroyed_cell = Cell::new(BaseTerrain::TowerDestroyed, dummy_position); // Updated call
        assert!(
            !tower_destroyed_cell.is_passable(),
            "TowerDestroyed should not be passable"
        );

        let tower_destroyed_with_content = Cell {
            position: dummy_position, // Added position
            base: BaseTerrain::TowerDestroyed,
            content: Some(CellContent::Champion(1, Team::Red)),
            animation: None,
        };
        assert!(
            !tower_destroyed_with_content.is_passable(),
            "TowerDestroyed with content should not be passable"
        );
    }

    #[test]
    fn test_encoded_cell_value_from_cell() {
        let dummy_position = (0, 0); // Dummy position for these tests

        // Test cases for different cell states
        let wall_cell = Cell::new(BaseTerrain::Wall, dummy_position); // Updated call
        assert_eq!(EncodedCellValue::from(&wall_cell), EncodedCellValue::Wall);

        let floor_cell = Cell::new(BaseTerrain::Floor, dummy_position); // Updated call
        assert_eq!(EncodedCellValue::from(&floor_cell), EncodedCellValue::Floor);

        let bush_cell = Cell::new(BaseTerrain::Bush, dummy_position); // Updated call
        assert_eq!(EncodedCellValue::from(&bush_cell), EncodedCellValue::Bush);

        let tower_destroyed_cell = Cell::new(BaseTerrain::TowerDestroyed, dummy_position); // Updated call
        assert_eq!(
            EncodedCellValue::from(&tower_destroyed_cell),
            EncodedCellValue::TowerDestroyed
        );

        let champion_cell = Cell {
            position: dummy_position, // Added position
            base: BaseTerrain::Floor, // Base shouldn't matter when content is present
            content: Some(CellContent::Champion(1, Team::Red)),
            animation: None,
        };
        assert_eq!(
            EncodedCellValue::from(&champion_cell),
            EncodedCellValue::Champion
        );

        let minion_cell = Cell {
            position: dummy_position, // Added position
            base: BaseTerrain::Wall,  // Base shouldn't matter when content is present
            content: Some(CellContent::Minion(1, Team::Red)),
            animation: None,
        };
        assert_eq!(
            EncodedCellValue::from(&minion_cell),
            EncodedCellValue::MinionRed
        );

        let flag_cell = Cell {
            position: dummy_position, // Added position
            base: BaseTerrain::Bush,  // Base shouldn't matter when content is present
            content: Some(CellContent::Flag(1, Team::Red)),
            animation: None,
        };
        assert_eq!(EncodedCellValue::from(&flag_cell), EncodedCellValue::Flag);

        let tower_cell = Cell {
            position: dummy_position,          // Added position
            base: BaseTerrain::TowerDestroyed, // Base shouldn't matter when content is present
            content: Some(CellContent::Tower(1, Team::Red)),
            animation: None,
        };
        assert_eq!(EncodedCellValue::from(&tower_cell), EncodedCellValue::Tower);

        let melee_animation_cell = Cell {
            position: dummy_position,                           // Added position
            base: BaseTerrain::Floor, // Base shouldn't matter when animation is present
            content: Some(CellContent::Champion(1, Team::Red)), // Content shouldn't matter when animation is present
            animation: Some(CellAnimation::MeleeHit),
        };
        assert_eq!(
            EncodedCellValue::from(&melee_animation_cell),
            EncodedCellValue::MeleeHitAnimation
        );

        let tower_hit_animation_cell = Cell {
            position: dummy_position, // Added position
            base: BaseTerrain::Wall,  // Base shouldn't matter when animation is present
            content: None,            // Content shouldn't matter when animation is present
            animation: Some(CellAnimation::TowerHit),
        };
        assert_eq!(
            EncodedCellValue::from(&tower_hit_animation_cell),
            EncodedCellValue::TowerHitAnimation
        );
    }
}

pub type PlayerId = usize;
pub type MinionId = usize;
pub type FlagId = usize;
pub type TowerId = usize;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum BaseTerrain {
    Wall,
    Floor,
    Bush,
    TowerDestroyed,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum CellContent {
    Champion(PlayerId),
    Minion(MinionId),
    Flag(FlagId),
    Tower(TowerId),
}

#[derive(Debug, Clone)]
pub struct Cell {
    pub base: BaseTerrain,
    pub content: Option<CellContent>,
}

impl Cell {
    pub fn new(base: BaseTerrain) -> Self {
        Cell {
            base,
            content: None,
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
    Minion = 5,
    Flag = 6,
    Tower = 7,
}

impl From<&Cell> for EncodedCellValue {
    fn from(cell: &Cell) -> Self {
        if let Some(content) = &cell.content {
            match content {
                CellContent::Champion(_) => EncodedCellValue::Champion,
                CellContent::Minion(_) => EncodedCellValue::Minion,
                CellContent::Flag(_) => EncodedCellValue::Flag,
                CellContent::Tower(_) => EncodedCellValue::Tower,
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

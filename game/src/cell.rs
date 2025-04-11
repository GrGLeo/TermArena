use crate::entities::player::PlayerId;
use crate::entities::minion::MinionId;

type FlagId = String;
type TowerId = String;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum BaseTerrain {
    Wall,
    Floor,
    Bush,
    TowerDestroyed,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum CellContent {
    Player(PlayerId),
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
        Cell{
            base,
            content: None
        }
    }

    pub fn is_passable(self) -> bool {
        match self.base {
            BaseTerrain::Wall => false,
            BaseTerrain::TowerDestroyed => false,
            BaseTerrain::Floor => self.content.is_none(),
            BaseTerrain::Bush => self.content.is_none(),
        }
    }
}

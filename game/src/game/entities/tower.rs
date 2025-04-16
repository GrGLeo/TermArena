use std::time::Duration;

use crate::game::cell::{TowerId, Cell, CellContent};
use crate::game::board::Board;

use super::{Fighter, Stats};

#[derive(Debug)]
pub struct Tower {
    pub tower_id: TowerId,
    pub stats: Stats,
    pub team_id: u8,
    pub row: u16,
    pub col: u16,
}

impl Tower {
    pub fn new(tower_id: TowerId, team_id: u8, row: u16, col: u16)  -> Self {
        let stats = Stats{
            attack_damage: 40,
            attack_speed: Duration::from_secs(3),
            health: 400,
            armor: 10,
        };

        Tower{
            tower_id,
            stats,
            team_id,
            row,
            col,
        }
    }

    pub fn place_tower(&self, board: &mut Board) {
        board.place_cell(CellContent::Tower(self.tower_id, self.team_id), self.row as usize, self.col as usize);
        board.place_cell(CellContent::Tower(self.tower_id, self.team_id), self.row as usize - 1, self.col as usize);
        board.place_cell(CellContent::Tower(self.tower_id, self.team_id), self.row as usize, self.col as usize + 1);
        board.place_cell(CellContent::Tower(self.tower_id, self.team_id), self.row as usize - 1, self.col as usize + 1);
    }
}

impl Fighter for Tower {
    fn take_damage(&mut self, damage: u8) {
        let reduced_damage = damage.saturating_sub(self.stats.armor);
        self.stats.health -= reduced_damage as u16;
    }

    fn attack(&self, target: &mut dyn Fighter) {
        target.take_damage(self.stats.attack_damage);
    }

    fn scan_range<'a>(&self, board: &'a Board) -> Vec<&'a Cell> {
        // range is implied here with: 6, 8
        let target_area = board.center_view(self.row, self.col, 6, 8);
        target_area.iter()
            .flat_map(|row| row.iter())
            .filter(|&&cell| {
                    if let Some(content) = &cell.content {
                        match content {
                            CellContent::Champion(_, team_id) | CellContent::Minion(_, team_id) => {
                                *team_id != self.team_id
                            },
                            _ => false,
                        }
                    } else {
                        false
                    }
            })
        .map(|&cell| cell)
        .collect()
    }
}

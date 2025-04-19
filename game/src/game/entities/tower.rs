use std::time::{Duration, Instant};

use crate::game::animation::melee::MeleeAnimation;
use crate::game::cell::{TowerId, Cell, CellContent};
use crate::game::board::Board;
use crate::game::BaseTerrain;

use super::{Fighter, Stats};

#[derive(Debug)]
pub struct Tower {
    pub tower_id: TowerId,
    pub team_id: u8,
    stats: Stats,
    destroyed: bool,
    last_attacked: Instant,
    row: u16,
    col: u16,
}

impl Tower {
    pub fn new(tower_id: TowerId, team_id: u8, row: u16, col: u16)  -> Self {
        let stats = Stats{
            attack_damage: 40,
            attack_speed: Duration::from_secs(3),
            health: 400,
            armor: 8,
        };

        Tower{
            tower_id,
            team_id,
            stats,
            destroyed: false,
            last_attacked: Instant::now(),
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

    pub fn is_destroyed(&self) -> bool {
        self.destroyed
    }

    pub fn destroy_tower(&self, board: &mut Board) {
        // Clear cell
        board.clear_cell(self.row as usize, self.col as usize);
        board.clear_cell(self.row as usize - 1, self.col as usize);
        board.clear_cell(self.row as usize, self.col as usize + 1);
        board.clear_cell(self.row as usize - 1, self.col as usize + 1);

        board.change_base(BaseTerrain::TowerDestroyed, self.row as usize, self.col as usize);
        board.change_base(BaseTerrain::TowerDestroyed, self.row as usize, self.col as usize + 1);
    }
}

impl Fighter for Tower {
    fn take_damage(&mut self, damage: u8) {
        let reduced_damage = damage.saturating_sub(self.stats.armor);
        self.stats.health = self.stats.health.saturating_sub(reduced_damage as u16);
        println!("Tower health: {}", self.stats.health);
        if self.stats.health == 0 {
            println!("Tower got 0 health");
            self.destroyed = true;
        }
    }

    fn can_attack(&mut self) -> Option<(u8, MeleeAnimation)> {
        if self.last_attacked + self.stats.attack_speed < Instant::now() {
            self.last_attacked = Instant::now();
            Some((self.stats.attack_damage, MeleeAnimation::new(1)))
        }
        else {
            None
        }
    }

    fn scan_range<'a>(&self, board: &'a Board) -> Option<&'a Cell> {
        // range is implied here with: 6, 8
        let target_area = board.center_view(self.row, self.col, 7, 9);
        let center_row = target_area.len() / 2;
        let center_col = target_area[0].len() / 2;

        target_area
            .iter()
            .enumerate()
            .flat_map(|(row_index, row)| {
                row.iter().enumerate().map(move |(col_index, cell)| (row_index, col_index, cell))
            })
        .filter_map(|(row, col, cell)| {
            if let Some(content) = &cell.content {
                match content {
                    CellContent::Champion(_, team_id ) | CellContent::Minion(_, team_id) => {
                        if *team_id != self.team_id {
                            Some((row, col, cell))
                        } else {
                            None
                        }
                    } 
                    _ => None
                }
            } else {
                None
            }
        })
        .min_by(|(r1, c1, _), (r2, c2, _)| {
            let dist1 = r1.abs_diff(center_row) + c1.abs_diff(center_col);
            let dist2 = r2.abs_diff(center_row) + c2.abs_diff(center_col);
            dist1.cmp(&dist2)
        })
        .map(|(_, _, &cell)| cell)
    }
}

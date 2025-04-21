use std::time::{Duration, Instant};

use crate::{errors::GameError, game::{animation::AnimationTrait, cell::Team, Board, MinionId}};

use super::{Fighter, Stats};

pub enum Lane {
    Top,
    Mid,
    Bottom,
}

pub struct Minion {
    pub minion_id: MinionId,
    pub team_id: Team,
    lane: Lane,
    stats: Stats,
    last_attacked: Instant,
    pub row: u16,
    pub col: u16,
}

impl Minion {
    pub fn new(minion_id: MinionId, team_id: Team, lane: Lane) -> Self {
        let stats = Stats {
            attack_damage: 6,
            attack_speed: Duration::from_millis(2500),
            health: 40,
            armor: 8,
        };

        let (row, col) = match team_id {
            Team::Blue => match lane {
                Lane::Top => (182, 4),
                Lane::Mid => (175, 24),
                Lane::Bottom => (194, 17),
            },
            Team::Red => match lane {
                Lane::Top => (4, 182),
                Lane::Mid => (24, 175),
                Lane::Bottom => (17, 194),
            },
        };

        Self {
            minion_id,
            team_id,
            lane,
            stats,
            last_attacked: Instant::now(),
            row,
            col,
        }
    }

    fn move_minion(
        &mut self,
        board: &mut Board,
        d_row: isize,
        d_col: isize,
    ) -> Result<(), GameError> {
        Ok(())
    }
}

impl Fighter for Minion {
    fn take_damage(&mut self, damage: u16) {
       todo!() 
    }

    fn can_attack(&mut self) -> Option<(u16, Box<dyn AnimationTrait>)> {
        todo!()
    }

    fn scan_range<'a>(&self, board: &'a crate::game::Board) -> Option<&'a crate::game::Cell> {
        todo!()
    }
}

use std::time::Duration;

use super::super::config;

#[derive(Debug, Copy, Clone)]
pub struct Position {
    pub row: u16,
    pub col: u16,
}

impl Position {
    pub fn new(row: u16, col: u16) -> Self {
        Position { row, col }
    }

    pub fn set(&mut self, row: u16, col: u16) {
        self.row = row;
        self.col = col;
    }

    pub fn move_up(&mut self) {
        if self.row > 0 {
            self.row -= 1
        }
    }

    pub fn move_down(&mut self) {
        if self.row < config::HEIGHT - 1 {
            self.row += 1
        }
    }

    pub fn move_left(&mut self) {
        if self.col > 0 {
            self.col -= 1
        }
    }

    pub fn move_right(&mut self) {
        if self.row < config::WIDTH - 1 {
            self.col += 1
        }
    }
}

#[derive(Debug)]
pub struct Stats {
    pub health: u8,
    pub attack_damage: u8,
    pub attack_speed: Duration,
    pub armor: u8,
}

impl Stats {
    pub fn default_player() -> Self {
        Stats{
            health: 80,
            attack_damage: 5,
            attack_speed: Duration::from_secs_f32(2.5),
            armor: 3
        }
    }

    pub fn default_minion() -> Self {
        Stats{
            health: 20,
            attack_damage: 4,
            attack_speed: Duration::from_secs_f32(3.),
            armor: 1
        }
    }
}



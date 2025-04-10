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
    attack_speed: f32,
    armor: u8,
}

use crate::errors::GameError;

use super::Board;

pub mod melee;

pub trait Animation: Send + Sync {
    fn get_id(&self) -> &usize;
    fn next(&mut self, row: u16, col: u16) -> Result<(u16, u16), GameError>;
    fn clean(&mut self, board: &mut Board);
    fn draw(&mut self, row: u16, col: u16, board: &mut Board) -> Result<(), GameError>;
}

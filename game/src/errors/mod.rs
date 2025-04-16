use thiserror::Error;

use crate::game::PlayerId;

#[derive(Debug, Error)]
pub enum GameError {
    #[error("Player: {0} cannot move there")]
    CannotMoveHere(PlayerId),
    #[error("Cell not found)")]
    NotFoundCell,
    #[error("Invalid input: {0}")]
    InvalidInput(String),
}

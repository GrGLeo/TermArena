use std::collections::HashMap;
use rand::prelude::*;
use strum::IntoEnumIterator;

use crate::errors::GameError;

use super::{cell::Team, entities::minion::{Lane, Minion}, Board, CellContent, MinionId};

#[derive(Debug)]
pub struct MinionManager {
    wave_creation: bool,
    minions_per_wave: u8,
    minions_this_wave: u8,
    minions: HashMap<MinionId, Minion>,
    ticker: u64,
}

impl MinionManager {
    pub fn new() -> Self {
        Self {
            wave_creation: false,
            minions_per_wave: 6,
            minions_this_wave: 0,
            minions: HashMap::new(),
            ticker: 0,

        }
    }

    pub fn make_wave(&mut self, board: &mut Board) {
        if (self.wave_creation && self.ticker % 2 == 0) || (self.ticker * 20 >= 10000 && self.ticker * 20 % 30000 == 0) {
            self.wave_creation = true;
            for team in Team::iter() {
                match team {
                    Team::Blue => {
                        for lane in Lane::iter() {
                            let minion_id = generate_minion_id().unwrap(); 
                            match lane {
                                Lane::Top => {
                                    let minion = Minion::new(minion_id, team, lane);
                                    board.place_cell(CellContent::Minion(minion_id, team), minion.row as usize, minion.col as usize);
                                    self.minions.insert(minion_id, minion);
                                }
                                Lane::Mid => {
                                    let minion = Minion::new(minion_id, team, lane);
                                    board.place_cell(CellContent::Minion(minion_id, team), minion.row as usize, minion.col as usize);
                                    self.minions.insert(minion_id, minion);
                                }
                                Lane::Bottom => {
                                    let minion = Minion::new(minion_id, team, lane);
                                    board.place_cell(CellContent::Minion(minion_id, team), minion.row as usize, minion.col as usize);
                                    self.minions.insert(minion_id, minion);
                                }
                            }
                        }
                    }
                    Team::Red => {
                        for lane in Lane::iter() {
                            let minion_id = generate_minion_id().unwrap(); 
                            match lane {
                                Lane::Top => {
                                    let minion = Minion::new(minion_id, team, lane);
                                    board.place_cell(CellContent::Minion(minion_id, team), minion.row as usize, minion.col as usize);
                                    self.minions.insert(minion_id, minion);
                                }
                                Lane::Mid => {
                                    let minion = Minion::new(minion_id, team, lane);
                                    board.place_cell(CellContent::Minion(minion_id, team), minion.row as usize, minion.col as usize);
                                    self.minions.insert(minion_id, minion);
                                }
                                Lane::Bottom => {
                                    let minion = Minion::new(minion_id, team, lane);
                                    board.place_cell(CellContent::Minion(minion_id, team), minion.row as usize, minion.col as usize);
                                    self.minions.insert(minion_id, minion);
                                }
                            }
                        }
                    }
                }
            }
            // Stopping wave creation
            self.minions_this_wave += 1;
            if self.minions_this_wave >= self.minions_per_wave {
                self.wave_creation = false
            }
        }
    }

    pub fn manage_minions_turn(&mut self, mut board: &mut Board) {
        self.minions.iter_mut().for_each(|(_, minion)| {
            minion.movement_phase(&mut board);
        });
        self.ticker += 1;
    }
}

fn generate_minion_id() -> Result<MinionId, GameError> {
    let mut rng = rand::rng();
    let nums: Vec<usize> = (1..99999).collect();
    if let Some(id) = nums.choose(&mut rng) {
        Ok(*id)
    } else {
        Err(GameError::GenerateIdError)
    }
}


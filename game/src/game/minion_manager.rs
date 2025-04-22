use rand::prelude::*;
use std::{collections::HashMap, time::{Duration, Instant}};
use strum::IntoEnumIterator;

use crate::errors::GameError;

use super::{
    Board, CellContent, MinionId,
    animation::AnimationTrait,
    cell::Team,
    entities::{
        Target,
        minion::{Lane, Minion},
    },
};

#[derive(Debug)]
pub struct MinionManager {
    minions_per_wave: u8,
    pub minions_this_wave: u8,
    pub minions: HashMap<MinionId, Minion>,
    pub wave_creation_time: Instant,
}

impl MinionManager {
    pub fn new() -> Self {
        Self {
            minions_per_wave: 6,
            minions_this_wave: 0,
            minions: HashMap::new(),
            wave_creation_time: Instant::now(),
        }
    }

    pub fn make_wave(&mut self, board: &mut Board) {
        let now = Instant::now();
        if now >= self.wave_creation_time  {
            for team in Team::iter() {
                match team {
                    Team::Blue => {
                        for lane in Lane::iter() {
                            let minion_id = generate_minion_id().unwrap();
                            match lane {
                                Lane::Top => {
                                    /*
                                    let minion = Minion::new(minion_id, team, lane);
                                    board.place_cell(
                                        CellContent::Minion(minion_id, team),
                                        minion.row as usize,
                                        minion.col as usize,
                                    );
                                    self.minions.insert(minion_id, minion);
                                    */
                                }
                                Lane::Mid => {
                                    /*
                                    let minion = Minion::new(minion_id, team, lane);
                                    board.place_cell(
                                        CellContent::Minion(minion_id, team),
                                        minion.row as usize,
                                        minion.col as usize,
                                    );
                                    self.minions.insert(minion_id, minion);
                                    */
                                }
                                Lane::Bottom => {
                                    let minion = Minion::new(minion_id, team, lane);
                                    board.place_cell(
                                        CellContent::Minion(minion_id, team),
                                        minion.row as usize,
                                        minion.col as usize,
                                    );
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
                                    /*
                                    let minion = Minion::new(minion_id, team, lane);
                                    board.place_cell(
                                        CellContent::Minion(minion_id, team),
                                        minion.row as usize,
                                        minion.col as usize,
                                    );
                                    self.minions.insert(minion_id, minion);
                                    */
                                }
                                Lane::Mid => {
                                    /*
                                    let minion = Minion::new(minion_id, team, lane);
                                    board.place_cell(
                                        CellContent::Minion(minion_id, team),
                                        minion.row as usize,
                                        minion.col as usize,
                                    );
                                    self.minions.insert(minion_id, minion);
                                    */
                                }
                                Lane::Bottom => {
                                    let minion = Minion::new(minion_id, team, lane);
                                    board.place_cell(
                                        CellContent::Minion(minion_id, team),
                                        minion.row as usize,
                                        minion.col as usize,
                                    );
                                    self.minions.insert(minion_id, minion);
                                }
                            }
                        }
                    }
                }
            }
            // Stopping wave creation
            self.minions_this_wave += 1;
            self.wave_creation_time = Instant::now() + Duration::from_millis(40);
            if self.minions_this_wave >= self.minions_per_wave {
                self.wave_creation_time = Instant::now() + Duration::from_secs(30);
                self.minions_this_wave = 0;
            }
        }
    }

    pub fn manage_minions_mouvements(&mut self, mut board: &mut Board) {
        self.minions.iter_mut().for_each(|(_, minion)| {
            minion.movement_phase(&mut board);
        });
    }

    pub fn manage_minions_attack(
        &mut self,
        mut board: &mut Board,
        new_animations: &mut Vec<Box<dyn AnimationTrait>>,
        pending_damages: &mut Vec<(Target, u16)>,
    ) {
        self.minions.iter_mut().for_each(|(_, minion)| {
            minion.attack_phase(&mut board, new_animations, pending_damages);
        });
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

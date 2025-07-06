use super::animation::{AnimationCommand, AnimationTrait};
use super::cell::{CellAnimation, Team};
use super::entities::Target;
use super::entities::projectile::{GameplayEffect, Projectile};
use super::{Board, CellContent};
use std::collections::HashMap;

pub struct ProjectileManager {
    projectiles: HashMap<u64, Projectile>,
    next_projectile_id: u64,
}

impl ProjectileManager {
    pub fn new() -> Self {
        ProjectileManager {
            projectiles: HashMap::new(),
            next_projectile_id: 0,
        }
    }

    pub fn create_projectile(
        &mut self,
        owner_id: u64,
        team_id: Team,
        start_pos: (u16, u16),
        end_pos: (u16, u16),
        speed: u32,
        payload: GameplayEffect,
        visual_cell_type: CellAnimation,
    ) {
        let id = self.next_projectile_id;
        self.next_projectile_id += 1;
        let projectile = Projectile::new(
            id,
            owner_id,
            team_id,
            start_pos,
            end_pos,
            speed,
            payload,
            visual_cell_type,
        );
        self.projectiles.insert(id, projectile);
    }

    pub fn update_and_check_collisions(
        &mut self,
        board: &Board,
    ) -> (Vec<Box<dyn AnimationTrait>>, Vec<(Target, u16)>, Vec<AnimationCommand>) {
        let mut animations_to_keep: Vec<Box<dyn AnimationTrait>> = Vec::new();
        let mut projectiles_to_remove: Vec<u64> = Vec::new();
        let mut pending_damages: Vec<(Target, u16)> = Vec::new();
        let mut animation_commands_executable: Vec<AnimationCommand> = Vec::new();

        for (id, projectile) in self.projectiles.iter_mut() {
            let command = projectile.next_frame(0, 0); // owner_row, owner_col not relevant for projectiles
            match command {
                AnimationCommand::Done => {
                    projectiles_to_remove.push(*id);
                }
                AnimationCommand::Draw {
                    row,
                    col,
                    animation_type,
                } => {
                    let mut hit_target = false;
                    if let Some(cell) = board.get_cell(row as usize, col as usize) {
                        match cell.content {
                            Some(CellContent::Champion(target_id, target_team)) => {
                                if projectile.team_id != target_team {
                                    // for now we only one GameplayEffect
                                    if let GameplayEffect::Damage(amount) = projectile.payload {
                                        pending_damages.push((Target::Champion(target_id), amount));
                                        hit_target = true;
                                    }
                                }
                            }
                            Some(CellContent::Minion(target_id, target_team)) => {
                                if projectile.team_id != target_team {
                                    // for now we only one GameplayEffect
                                    if let GameplayEffect::Damage(amount) = projectile.payload {
                                        pending_damages.push((Target::Minion(target_id), amount));
                                        hit_target = true;
                                    }
                                }
                            }
                            _ => {},
                        }
                    }

                    if hit_target {
                        projectiles_to_remove.push(*id);
                        animation_commands_executable.push(AnimationCommand::Clear { row, col });
                    } else {
                        animations_to_keep.push(Box::new(projectile.clone()));
                        animation_commands_executable.push(AnimationCommand::Draw { row, col, animation_type });
                    }
                }
                _ => {},
            }
        }

        for id in projectiles_to_remove {
            self.projectiles.remove(&id);
        }

        (animations_to_keep, pending_damages, animation_commands_executable)
    }
}

#[cfg(test)]
mod tests {
    use super::super::cell::CellAnimation;
    use super::*;
    use crate::game::cell::Team;

    fn create_dummy_board(rows: usize, cols: usize) -> Board {
        Board::new(rows, cols)
    }

    #[test]
    fn test_projectile_manager_creation() {
        let manager = ProjectileManager::new();
        assert_eq!(manager.projectiles.len(), 0);
        assert_eq!(manager.next_projectile_id, 0);
    }

    #[test]
    fn test_create_projectile() {
        let mut manager = ProjectileManager::new();
        let start_pos = (0, 0);
        let end_pos = (10, 10);
        let speed = 1;
        let payload = GameplayEffect::Damage(10);
        let visual_type = CellAnimation::Projectile;

        manager.create_projectile(
            1,
            Team::Blue,
            start_pos,
            end_pos,
            speed,
            payload,
            visual_type,
        );

        assert_eq!(manager.projectiles.len(), 1);
        assert_eq!(manager.next_projectile_id, 1);
        let projectile = manager.projectiles.get(&0).unwrap();
        assert_eq!(projectile.owner_id, 1);
        assert_eq!(projectile.path[0], start_pos);
    }

    #[test]
    fn test_update_and_check_collisions_projectile_finishes() {
        let mut manager = ProjectileManager::new();
        let board = create_dummy_board(5, 5);
        let start_pos = (0, 0);
        let end_pos = (1, 0);
        let speed = 1; // Moves every tick
        let payload = GameplayEffect::Damage(10);
        let visual_type = CellAnimation::Projectile;

        manager.create_projectile(
            1,
            Team::Blue,
            start_pos,
            end_pos,
            speed,
            payload,
            visual_type,
        );

        // Tick 1: Projectile moves to start_pos
        let _ = manager.update_and_check_collisions(&board);
        assert_eq!(manager.projectiles.len(), 1);
        // TODO: Assert on animations returned

        // Tick 2: Projectile moves to end_pos
        let _ = manager.update_and_check_collisions(&board);
        assert_eq!(manager.projectiles.len(), 1);

        // Tick 3: Projectile finishes
        let _ = manager.update_and_check_collisions(&board);
        assert_eq!(manager.projectiles.len(), 0);
    }
}

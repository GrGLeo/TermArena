use crate::game::spell::ProjectileType;

use super::animation::{AnimationCommand, AnimationTrait};
use super::cell::{CellAnimation, Team};
use super::entities::Target;
use super::entities::minion::Minion;
use super::entities::projectile::{GameplayEffect, PathingLogic, Projectile};
use super::entities::tower::Tower;
use super::spell::ProjectileBlueprint;
use super::{Board, CellContent, Champion, MinionId, PlayerId, TowerId};
use std::collections::HashMap;

#[derive(Debug)]
pub struct ProjectileManager {
    pub projectiles: HashMap<u64, Projectile>,
    next_projectile_id: u64,
}

impl ProjectileManager {
    pub fn new() -> Self {
        ProjectileManager {
            projectiles: HashMap::new(),
            next_projectile_id: 0,
        }
    }

    pub fn create_from_blueprint(&mut self, blueprint: ProjectileBlueprint) {
        match blueprint.projectile_type {
            ProjectileType::LockOn => {
                if let Some(target_id) = blueprint.target_id {
                    self.create_homing_projectile(
                        blueprint.owner_id,
                        blueprint.team_id,
                        target_id,
                        blueprint.start_pos,
                        blueprint.speed,
                        blueprint.payload,
                        blueprint.visual_cell_type,
                    );
                }
            }
            ProjectileType::SkillShot => {
                self.create_skillshot_projectile(
                    blueprint.owner_id,
                    blueprint.team_id,
                    blueprint.start_pos,
                    blueprint.end_pos,
                    blueprint.speed,
                    blueprint.payload,
                    blueprint.visual_cell_type,
                );
            }
        }
    }

    pub fn create_skillshot_projectile(
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
        let projectile = Projectile::from_skillshot(
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

    pub fn create_homing_projectile(
        &mut self,
        owner_id: u64,
        team_id: Team,
        target_id: Target,
        start_pos: (u16, u16),
        speed: u32,
        payload: GameplayEffect,
        visual_cell_type: CellAnimation,
    ) {
        let id = self.next_projectile_id;
        self.next_projectile_id += 1;
        let projectile = Projectile::from_homing_shot(
            id,
            owner_id,
            team_id,
            start_pos,
            target_id,
            speed,
            payload,
            visual_cell_type,
        );
        self.projectiles.insert(id, projectile);
    }

    pub fn update_and_check_collisions(
        &mut self,
        board: &Board,
        champions: &HashMap<PlayerId, Champion>,
        minions: &HashMap<MinionId, Minion>,
        towers: &HashMap<TowerId, Tower>,
    ) -> (Vec<(Target, GameplayEffect)>, Vec<AnimationCommand>) {
        let mut projectiles_to_remove: Vec<u64> = Vec::new();
        let mut pending_effects: Vec<(Target, GameplayEffect)> = Vec::new();
        let mut animation_commands_executable: Vec<AnimationCommand> = Vec::new();

        for (id, projectile) in self.projectiles.iter_mut() {
            let (target_row, target_col) = match &projectile.pathing {
                PathingLogic::Straight { .. } => (0, 0),
                PathingLogic::LockOn { target_id } => match target_id {
                    Target::Champion(id) => {
                        if let Some(champion) = champions.get(id) {
                            (champion.row, champion.col)
                        } else {
                            projectiles_to_remove.push(*id as u64);
                            continue;
                        }
                    }
                    Target::Minion(id) => {
                        if let Some(minion) = minions.get(id) {
                            (minion.row, minion.col)
                        } else {
                            projectiles_to_remove.push(*id as u64);
                            continue;
                        }
                    }
                    Target::Tower(id) => {
                        if let Some(tower) = towers.get(id) {
                            (tower.row, tower.col)
                        } else {
                            projectiles_to_remove.push(*id as u64);
                            continue;
                        }
                    }
                    _ => {
                        projectiles_to_remove.push(*id as u64);
                        continue;
                    }
                },
            };

            if let Some(last_pos) = projectile.get_last_drawn_pos() {
                animation_commands_executable.push(AnimationCommand::Clear {
                    row: last_pos.0,
                    col: last_pos.1,
                });
            }

            let command = projectile.next_frame(target_row, target_col);
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
                                hit_target = add_damages(
                                    &mut pending_effects,
                                    Target::Champion(target_id),
                                    &projectile.payload,
                                    projectile.team_id,
                                    target_team,
                                )
                            }
                            Some(CellContent::Minion(target_id, target_team)) => {
                                hit_target = add_damages(
                                    &mut pending_effects,
                                    Target::Minion(target_id),
                                    &projectile.payload,
                                    projectile.team_id,
                                    target_team,
                                )
                            }
                            Some(CellContent::Tower(target_id, target_team)) => {
                                hit_target = add_damages(
                                    &mut pending_effects,
                                    Target::Tower(target_id),
                                    &projectile.payload,
                                    projectile.team_id,
                                    target_team,
                                )
                            }
                            _ => {}
                        }
                    }

                    if hit_target {
                        projectiles_to_remove.push(*id);
                        animation_commands_executable.push(AnimationCommand::Clear { row, col });
                    } else {
                        animation_commands_executable.push(AnimationCommand::Draw {
                            row,
                            col,
                            animation_type,
                        });
                    }
                }
                _ => {}
            }
        }

        for id in projectiles_to_remove {
            self.projectiles.remove(&id);
        }

        (pending_effects, animation_commands_executable)
    }
}

fn add_damages(
    pending_damages: &mut Vec<(Target, GameplayEffect)>,
    target: Target,
    payload: &GameplayEffect,
    team: Team,
    target_team: Team,
) -> bool {
    if team != target_team {
        match payload {
            GameplayEffect::Damage(..) => {
                pending_damages.push((target, *payload));
                return true;
            }
            GameplayEffect::Stun(..) => {
                pending_damages.push((target, *payload));
                return true;
            }
        }
    }
    false
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::config::{ChampionStats, TowerStats};
    use crate::game::cell::{CellAnimation, Team};
    use crate::game::entities::champion::Champion;
    use crate::game::entities::projectile::PathingLogic;
    use crate::game::entities::tower::Tower;
    use crate::game::{PlayerId, TowerId};

    fn create_dummy_board(rows: usize, cols: usize) -> Board {
        Board::new(rows, cols)
    }

    fn mock_champion_stats() -> ChampionStats {
        ChampionStats {
            attack_damage: 50,
            attack_speed_ms: 1000,
            health: 500,
            armor: 10,
            xp_per_level: vec![100, 200],
            level_up_health_increase: 50,
            level_up_attack_damage_increase: 5,
            level_up_armor_increase: 2,
            attack_range_row: 3,
            attack_range_col: 3,
        }
    }

    fn mock_tower_stats() -> TowerStats {
        TowerStats {
            attack_damage: 100,
            attack_speed_secs: 2,
            health: 1000,
            armor: 20,
            attack_range_row: 7,
            attack_range_col: 9,
        }
    }

    #[test]
    fn test_create_skillshot_projectile() {
        let mut manager = ProjectileManager::new();
        manager.create_skillshot_projectile(
            101,
            Team::Blue,
            (10, 10),
            (20, 20),
            1,
            GameplayEffect::Damage(50),
            CellAnimation::Projectile,
        );
        assert_eq!(manager.projectiles.len(), 1);
        let projectile = manager.projectiles.get(&0).unwrap();
        assert!(matches!(projectile.pathing, PathingLogic::Straight { .. }));
    }

    #[test]
    fn test_create_homing_projectile() {
        let mut manager = ProjectileManager::new();
        manager.create_homing_projectile(
            102,
            Team::Red,
            Target::Champion(202),
            (5, 5),
            2,
            GameplayEffect::Damage(30),
            CellAnimation::Projectile,
        );
        assert_eq!(manager.projectiles.len(), 1);
        let projectile = manager.projectiles.get(&0).unwrap();
        assert!(matches!(projectile.pathing, PathingLogic::LockOn { .. }));
    }

    #[test]
    fn test_create_lockon_from_blueprint() {
        let mut manager = ProjectileManager::new();
        let blueprint = ProjectileBlueprint {
            projectile_type: ProjectileType::LockOn,
            owner_id: 101,
            team_id: Team::Blue,
            target_id: Option::Some(Target::Minion(5)),
            start_pos: (0, 0),
            end_pos: (10, 10),
            speed: 2,
            payload: GameplayEffect::Damage(5),
            visual_cell_type: CellAnimation::Projectile,
        };
        manager.create_from_blueprint(blueprint);
        assert_eq!(manager.projectiles.len(), 1);
        let projectile = manager.projectiles.get(&0).unwrap();
        assert!(matches!(projectile.pathing, PathingLogic::LockOn { .. }));
    }

    #[test]
    fn test_create_skillshot_from_blueprint() {
        let mut manager = ProjectileManager::new();
        let blueprint = ProjectileBlueprint {
            projectile_type: ProjectileType::SkillShot,
            owner_id: 101,
            team_id: Team::Blue,
            target_id: Option::Some(Target::Minion(5)),
            start_pos: (0, 0),
            end_pos: (10, 10),
            speed: 2,
            payload: GameplayEffect::Damage(5),
            visual_cell_type: CellAnimation::Projectile,
        };
        manager.create_from_blueprint(blueprint);
        assert_eq!(manager.projectiles.len(), 1);
        let projectile = manager.projectiles.get(&0).unwrap();
        assert!(matches!(projectile.pathing, PathingLogic::Straight { .. }));
    }

    #[test]
    fn test_update_skillshot_misses_and_finishes() {
        let mut manager = ProjectileManager::new();
        let board = create_dummy_board(20, 20);
        let champions = HashMap::<PlayerId, Champion>::new();
        let minions = HashMap::new();
        let towers = HashMap::<TowerId, Tower>::new();

        manager.create_skillshot_projectile(
            101,
            Team::Blue,
            (0, 0),
            (2, 0),
            1,
            GameplayEffect::Damage(10),
            CellAnimation::Projectile,
        );

        for _ in 0..3 {
            let (pending_damages, _) =
                manager.update_and_check_collisions(&board, &champions, &minions, &towers);
            println!("{:?}", pending_damages);
            assert!(pending_damages.is_empty());
            assert_eq!(manager.projectiles.len(), 1);
        }

        let (pending_damages, _) =
            manager.update_and_check_collisions(&board, &champions, &minions, &towers);
        assert!(pending_damages.is_empty());
        assert!(manager.projectiles.is_empty());
    }

    #[test]
    fn test_update_projectile_hits_champion() {
        let mut manager = ProjectileManager::new();
        let mut board = create_dummy_board(20, 20);
        let mut champions = HashMap::new();
        let minions = HashMap::new();
        let towers = HashMap::new();

        let target_id = 202;
        let target_pos = (10, 12);
        let target_champion = Champion::new(
            target_id,
            Team::Red,
            target_pos.0,
            target_pos.1,
            mock_champion_stats(),
            HashMap::new(),
        );
        champions.insert(target_id, target_champion);
        board.place_cell(
            CellContent::Champion(target_id, Team::Red),
            target_pos.0 as usize,
            target_pos.1 as usize,
        );

        manager.create_skillshot_projectile(
            101,
            Team::Blue,
            (10, 10),
            target_pos,
            1,
            GameplayEffect::Damage(50),
            CellAnimation::Projectile,
        );

        // Tick 1 & 2: Projectile moves closer
        println!("{:?}", manager);
        manager.update_and_check_collisions(&board, &champions, &minions, &towers);
        println!("{:?}", manager);
        manager.update_and_check_collisions(&board, &champions, &minions, &towers);
        println!("{:?}", manager);

        // Tick 3: Projectile should hit the target
        let (damages, _) =
            manager.update_and_check_collisions(&board, &champions, &minions, &towers);
        assert_eq!(damages.len(), 1);
        assert!(matches!(damages[0], (Target::Champion(id), GameplayEffect::Damage(50)) if id == target_id));
        assert!(manager.projectiles.is_empty());
    }

    #[test]
    fn test_update_projectile_hits_tower() {
        let mut manager = ProjectileManager::new();
        let mut board = create_dummy_board(20, 20);
        let champions = HashMap::new();
        let minions = HashMap::new();
        let mut towers = HashMap::new();

        let target_id = 303 as TowerId;
        let target_pos = (0, 5);
        let target_tower = Tower::new(
            target_id,
            Team::Red,
            target_pos.0,
            target_pos.1,
            mock_tower_stats(),
        );
        towers.insert(target_id, target_tower);
        board.place_cell(
            CellContent::Tower(target_id, Team::Red),
            target_pos.0 as usize,
            target_pos.1 as usize,
        );

        manager.create_homing_projectile(
            101,
            Team::Blue,
            Target::Tower(target_id),
            (0, 2),
            1,
            GameplayEffect::Damage(50),
            CellAnimation::Projectile,
        );

        // Tick 1, 2, 3: Move closer
        manager.update_and_check_collisions(&board, &champions, &minions, &towers);
        manager.update_and_check_collisions(&board, &champions, &minions, &towers);
        let (damages, _) =
            manager.update_and_check_collisions(&board, &champions, &minions, &towers);

        assert_eq!(damages.len(), 1);
        assert!(matches!(damages[0], (Target::Tower(id), GameplayEffect::Damage(50)) if id == target_id));
        assert!(manager.projectiles.is_empty());
    }

    #[test]
    fn test_update_homing_projectile_tracks_target() {
        let mut manager = ProjectileManager::new();
        let board = create_dummy_board(20, 20);
        let mut champions = HashMap::new();
        let minions = HashMap::new();
        let towers = HashMap::new();

        let target_id = 202;
        let target_champion = Champion::new(
            target_id,
            Team::Red,
            10,
            13,
            mock_champion_stats(),
            HashMap::new(),
        );
        champions.insert(target_id, target_champion);

        manager.create_homing_projectile(
            102,
            Team::Blue,
            Target::Champion(target_id),
            (10, 10),
            1,
            GameplayEffect::Damage(30),
            CellAnimation::Projectile,
        );

        // Tick 1: Projectile moves towards (10, 13)
        manager.update_and_check_collisions(&board, &champions, &minions, &towers);
        let proj1 = manager.projectiles.get(&0).unwrap();
        assert_eq!(proj1.current_position, (10, 11));

        // Move the target
        champions.get_mut(&target_id).unwrap().row = 11;
        champions.get_mut(&target_id).unwrap().col = 14;

        // Tick 2: Projectile should now move towards the new position (11, 14)
        manager.update_and_check_collisions(&board, &champions, &minions, &towers);
        let proj2 = manager.projectiles.get(&0).unwrap();
        assert_eq!(proj2.current_position, (11, 12)); // Moves diagonally
    }
}

use std::time::{Duration, Instant}; // Add this import

use crate::game::cell::Team;

#[derive(Debug)] // Add Debug derive for easier debugging
pub struct AoeSpell {
    pub team_id: Team,
    pub caster_row: u16, // Renamed for clarity
    pub caster_col: u16, // Renamed for clarity
    pub damage: u16,
    pub max_radius: u8,
    current_tick_level: u8, // Renamed to avoid confusion with game ticks
    last_damage_time: Instant, // Use Instant for timing
    tick_interval: Duration, // Use Duration for interval
}

impl AoeSpell {
    pub fn new(team_id: Team, caster_pos: (u16, u16)) -> AoeSpell {
        let (caster_row, caster_col) = caster_pos; // Destructure for clarity
        AoeSpell {
            team_id,
            caster_row,
            caster_col,
            damage: 25,
            max_radius: 3,
            current_tick_level: 0, // Start at 0, first tick will be level 1
            last_damage_time: Instant::now(),
            tick_interval: Duration::from_millis(200), // 200ms per tick
        }
    }

    pub fn next_tick(&mut self) -> Option<Vec<(u16, u16)>> {
        // Check if enough time has passed for the next tick
        if self.last_damage_time.elapsed() < self.tick_interval {
            return None; // Not time for the next tick yet
        }

        self.current_tick_level += 1; // Advance to the next tick level
        self.last_damage_time = Instant::now(); // Reset timer for the next tick

        if self.current_tick_level > self.max_radius {
            return None; // Spell is finished
        }

        let mut affected_cells = Vec::new();
        let current_radius = self.current_tick_level as i16; // Convert to signed for calculations

        // Generate cells in a ring at the current_radius
        for i in -current_radius..=current_radius {
            let j_abs = current_radius - i.abs(); // Calculate the absolute j-coordinate

            // Add cells for (i, j_abs) and (i, -j_abs)
            affected_cells.push((
                (self.caster_row as i16 + i) as u16,
                (self.caster_col as i16 + j_abs) as u16,
            ));
            if j_abs != 0 { // Avoid duplicating the center line if j_abs is 0
                affected_cells.push((
                    (self.caster_row as i16 + i) as u16,
                    (self.caster_col as i16 - j_abs) as u16,
                ));
            }
        }
        Some(affected_cells)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::game::cell::Team; // Assuming Team enum is accessible

    #[test]
    fn test_aoe_spell_new() {
        let team_id = Team::Blue;
        let champion_pos = (10, 20);
        let spell = AoeSpell::new(team_id, champion_pos);

        assert_eq!(spell.team_id, team_id);
        assert_eq!(spell.caster_row, 10);
        assert_eq!(spell.caster_col, 20);
        assert_eq!(spell.damage, 25);
        assert_eq!(spell.max_radius, 3);
        assert_eq!(spell.current_tick_level, 0);
        // last_damage_time is Instant::now(), so we can't assert exact equality.
        // We can check if it's roughly now, but for unit tests, it's often skipped
        // or mocked if precise time control is needed.
        assert_eq!(spell.tick_interval, Duration::from_millis(200));
    }

    #[test]
    fn test_aoe_spell_next_tick_timing_and_completion() {
        let team_id = Team::Red;
        let champion_pos = (50, 50);
        let mut spell = AoeSpell::new(team_id, champion_pos);

        // Initially, no tick should occur immediately
        assert!(spell.next_tick().is_none(), "next_tick should return None immediately after creation");

        // Manually advance time for the first tick
        spell.last_damage_time = Instant::now() - spell.tick_interval - Duration::from_millis(1);
        let first_tick_cells = spell.next_tick();
        assert!(first_tick_cells.is_some(), "First tick should occur after interval");
        assert_eq!(spell.current_tick_level, 1, "current_tick_level should be 1 after first tick");

        // Second tick - not enough time passed (because last_damage_time was just reset)
        assert!(spell.next_tick().is_none(), "next_tick should return None if not enough time passed");

        // Manually advance time for second tick
        spell.last_damage_time = Instant::now() - spell.tick_interval - Duration::from_millis(1);
        let second_tick_cells = spell.next_tick();
        assert!(second_tick_cells.is_some(), "Second tick should occur after interval");
        assert_eq!(spell.current_tick_level, 2, "current_tick_level should be 2 after second tick");

        // Manually advance time for third tick
        spell.last_damage_time = Instant::now() - spell.tick_interval - Duration::from_millis(1);
        let third_tick_cells = spell.next_tick();
        assert!(third_tick_cells.is_some(), "Third tick should occur after interval");
        assert_eq!(spell.current_tick_level, 3, "current_tick_level should be 3 after third tick");

        // After max_radius (3) ticks, the spell should be done
        spell.last_damage_time = Instant::now() - spell.tick_interval - Duration::from_millis(1);
        assert!(spell.next_tick().is_none(), "Spell should be finished after max_radius ticks");
        assert_eq!(spell.current_tick_level, 4, "current_tick_level should be > max_radius when finished");
    }

    #[test]
    fn test_aoe_spell_next_tick_affected_cells() {
        let team_id = Team::Blue;
        let caster_pos = (10, 10);
        let mut spell = AoeSpell::new(team_id, caster_pos);

        // Simulate time passing for each tick
        // Tick 1 (Radius 1)
        spell.last_damage_time = Instant::now() - spell.tick_interval - Duration::from_millis(1);
        let cells_r1 = spell.next_tick().unwrap();
        let mut expected_r1 = vec![
            (10, 11), (11, 10), (10, 9), (9, 10) // Manhattan distance 1 from (10,10)
        ];
        expected_r1.sort_unstable(); // Sort for consistent comparison
        let mut actual_r1 = cells_r1;
        actual_r1.sort_unstable();
        assert_eq!(actual_r1, expected_r1, "Cells for radius 1 are incorrect");

        // Tick 2 (Radius 2)
        spell.last_damage_time = Instant::now() - spell.tick_interval - Duration::from_millis(1);
        let cells_r2 = spell.next_tick().unwrap();
        let mut expected_r2 = vec![
            (10, 12), (11, 11), (12, 10), (11, 9), // Manhattan distance 2 from (10,10)
            (10, 8), (9, 9), (8, 10), (9, 11)
        ];
        expected_r2.sort_unstable();
        let mut actual_r2 = cells_r2;
        actual_r2.sort_unstable();
        assert_eq!(actual_r2, expected_r2, "Cells for radius 2 are incorrect");

        // Tick 3 (Radius 3)
        spell.last_damage_time = Instant::now() - spell.tick_interval - Duration::from_millis(1);
        let cells_r3 = spell.next_tick().unwrap();
        let mut expected_r3 = vec![
            (10, 13), (11, 12), (12, 11), (13, 10), // Manhattan distance 3 from (10,10)
            (12, 9), (11, 8), (10, 7), (9, 8),
            (8, 9), (7, 10), (8, 11), (9, 12)
        ];
        expected_r3.sort_unstable();
        let mut actual_r3 = cells_r3;
        actual_r3.sort_unstable();
        assert_eq!(actual_r3, expected_r3, "Cells for radius 3 are incorrect");

        // After max_radius, should return None
        spell.last_damage_time = Instant::now() - spell.tick_interval - Duration::from_millis(1);
        assert!(spell.next_tick().is_none(), "Should return None after max_radius ticks");
    }
}
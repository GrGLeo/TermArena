use super::Board;
use std::collections::{BinaryHeap, HashMap, HashSet, VecDeque};

pub fn get_valid_neighbors(board: &Board, row: u16, col: u16) -> Vec<(u16, u16)> {
    let mut valid_neighbors: Vec<(u16, u16)> = Vec::new();
    for i in -1..=1 {
        for j in -1..=1 {
            if i == 0 && j == 0 {
                continue;
            }
            let neighbors_row = row as isize + i;
            let neighbors_col = col as isize + j;
            if let Some(cell) = board.get_cell(neighbors_row as usize, neighbors_col as usize) {
                if cell.is_passable() {
                    valid_neighbors.push((neighbors_row as u16, neighbors_col as u16))
                }
            }
        }
    }
    valid_neighbors
}

pub fn calculate_heuristic(row: u16, col: u16, goal_row: u16, goal_col: u16) -> u16 {
    (row.abs_diff(goal_row)).max(col.abs_diff(goal_col))
}

pub fn is_adjacent_to_goal(pos: (u16, u16), goal: (u16, u16)) -> bool {
    let (r1, c1) = pos;
    let (r2, c2) = goal;
    let dr = (r1 as i16 - r2 as i16).abs();
    let dc = (c1 as i16 - c2 as i16).abs();
    (dr <= 1 && dc <= 1) && (dr > 0 || dc > 0)
}

#[derive(Clone, Copy, Eq, PartialEq, Hash)]
pub struct PathNode {
    position: (u16, u16),
    g_cost: u16,
    h_cost: u16,
    f_cost: u16,
}

impl Ord for PathNode {
    fn cmp(&self, other: &Self) -> std::cmp::Ordering {
        other
            .f_cost
            .cmp(&self.f_cost)
            .then_with(|| self.g_cost.cmp(&other.g_cost))
    }
}

impl PartialOrd for PathNode {
    fn partial_cmp(&self, other: &Self) -> Option<std::cmp::Ordering> {
        Some(self.cmp(other))
    }
}

pub fn find_path_on_board(
    board: &Board,
    start: (u16, u16),
    goal: (u16, u16),
) -> Option<VecDeque<(u16, u16)>> {
    if start == goal {
        let mut path = VecDeque::new();
        path.push_front(start);
        return Some(path);
    }
    if is_adjacent_to_goal(start, goal) {
        let mut path = VecDeque::new();
        path.push_front(start);
        return Some(path);
    }

    let mut open_set: BinaryHeap<PathNode> = BinaryHeap::new();
    let mut closed_set: HashSet<(u16, u16)> = HashSet::new();
    let mut g_costs: HashMap<(u16, u16), u16> = HashMap::new();
    let mut parents: HashMap<(u16, u16), (u16, u16)> = HashMap::new();

    let start_node_g_cost = 0;
    let start_node_h_cost = calculate_heuristic(start.0, start.1, goal.0, goal.1);
    let start_node_f_cost = start_node_g_cost + start_node_h_cost;

    let start_node = PathNode {
        position: start,
        g_cost: start_node_g_cost,
        h_cost: start_node_h_cost,
        f_cost: start_node_f_cost,
    };

    open_set.push(start_node);
    g_costs.insert(start, start_node_g_cost);


    while let Some(current_node) = open_set.pop() {

        if closed_set.contains(&current_node.position) {
            continue;
        }

        if is_adjacent_to_goal(current_node.position, goal) {
            let mut path = VecDeque::new();
            let mut current_pos = current_node.position;
            while current_pos != start {
                path.push_front(current_pos);
                current_pos = *parents
                    .get(&current_pos)
                    .expect("Path reconstruction error: missing parent");
            }

            return Some(path);
        }

        closed_set.insert(current_node.position);

        let neighbors_pos =
            get_valid_neighbors(board, current_node.position.0, current_node.position.1);

        for neighbor_pos in neighbors_pos {

            if closed_set.contains(&neighbor_pos) {
                continue;
            }

            let tentative_g_cost = current_node.g_cost + 1;

            match g_costs.get(&neighbor_pos) {
                Some(existing_g_cost) => {
                    if tentative_g_cost < *existing_g_cost {
                        parents.insert(neighbor_pos, current_node.position);
                        g_costs.insert(neighbor_pos, tentative_g_cost);

                        let neighbor_h_cost =
                            calculate_heuristic(neighbor_pos.0, neighbor_pos.1, goal.0, goal.1);
                        let neighbor_f_cost = tentative_g_cost + neighbor_h_cost;

                        let neighbor_node = PathNode {
                            position: neighbor_pos,
                            g_cost: tentative_g_cost,
                            h_cost: neighbor_h_cost,
                            f_cost: neighbor_f_cost,
                        };
                        open_set.push(neighbor_node);
                    } else {
                    }
                }
                None => {
                    parents.insert(neighbor_pos, current_node.position);
                    g_costs.insert(neighbor_pos, tentative_g_cost);

                    let neighbor_h_cost =
                        calculate_heuristic(neighbor_pos.0, neighbor_pos.1, goal.0, goal.1);
                    let neighbor_f_cost = tentative_g_cost + neighbor_h_cost;

                    let neighbor_node = PathNode {
                        position: neighbor_pos,
                        g_cost: tentative_g_cost,
                        h_cost: neighbor_h_cost,
                        f_cost: neighbor_f_cost,
                    };
                    open_set.push(neighbor_node);
                }
            }
        }
    }

    None
}

#[cfg(test)]
mod pathfinding_tests {
    // You might want to name this module appropriately

    use super::*;
    use crate::game::board::Board;
    use crate::game::cell::{BaseTerrain, CellContent, Team};

    #[test]
    fn test_get_valid_neighbors_middle_of_open_board() {
        // Create a small board with no obstacles
        let board = Board::new(5, 5); // 5x5 board

        let center_row = 2;
        let center_col = 2;

        // Expected neighbors for a cell in the middle of an open board (assuming 8-directional movement)
        let mut expected_neighbors = vec![
            (1, 2), // Up
            (3, 2), // Down
            (2, 1), // Left
            (2, 3), // Right
            (1, 1), // Up-Left
            (1, 3), // Up-Right
            (3, 1), // Down-Left
            (3, 3), // Down-Right
        ];
        expected_neighbors.sort(); // Sort for reliable comparison

        // Call the function you need to implement
        let mut actual_neighbors = get_valid_neighbors(&board, center_row, center_col);
        actual_neighbors.sort();

        assert_eq!(
            actual_neighbors, expected_neighbors,
            "Neighbors in the middle of an open board should include all 8 adjacent cells."
        );
    }

    #[test]
    fn test_get_valid_neighbors_near_edge_with_obstacle() {
        // Create a board with obstacles and edges
        let mut board = Board::new(5, 5); // 5x5 board

        let test_row = 0; // Top edge
        let test_col = 1;

        // Place a wall to the right
        board.change_base(
            BaseTerrain::Wall,
            test_row as usize,
            (test_col + 1) as usize,
        );

        // Place a minion below and to the left (making that cell impassable)
        board.place_cell(
            CellContent::Minion(1, Team::Blue),
            (test_row + 1) as usize,
            (test_col - 1) as usize,
        );

        // Expected valid neighbors for (0, 1) considering the edge, wall, and minion
        // Possible neighbors are: (0,0), (0,2), (1,0), (1,1), (1,2)
        // (0,0) is valid (Left)
        // (0,2) is blocked by Wall (Right) - Invalid
        // (1,0) is blocked by Minion (Down-Left) - Invalid
        // (1,1) is valid (Down)
        // (1,2) is valid (Down-Right)
        let mut expected_neighbors = vec![
            (0, 0), // Left
            (1, 1), // Down
            (1, 2), // Down-Right
        ];
        expected_neighbors.sort(); // Sort for reliable comparison

        // Call the function you need to implement
        let mut actual_neighbors = get_valid_neighbors(&board, test_row, test_col);
        actual_neighbors.sort();

        assert_eq!(
            actual_neighbors, expected_neighbors,
            "Neighbors near the edge and next to obstacles should be correctly identified."
        );
    }

    #[test]
    fn test_heuristic_same_cell() {
        let start = (10, 10);
        let goal = (10, 10);
        let expected_heuristic: u16 = 0; // The cost to get from a cell to itself is 0.

        let actual_heuristic = calculate_heuristic(start.0, start.1, goal.0, goal.1);

        // Using assert_eq! for integer comparison
        assert_eq!(
            actual_heuristic, expected_heuristic,
            "Heuristic for the same cell should be 0."
        );
    }

    #[test]
    fn test_heuristic_adjacent_horizontal() {
        let start = (10, 10);
        let goal = (10, 11);
        // Diagonal Distance: max(|10-10|, |10-11|) = max(0, 1) = 1
        let expected_heuristic: u16 = 1;

        let actual_heuristic = calculate_heuristic(start.0, start.1, goal.0, goal.1);

        assert_eq!(
            actual_heuristic, expected_heuristic,
            "Heuristic for adjacent horizontal cells should be 1."
        );
    }

    #[test]
    fn test_heuristic_adjacent_diagonal() {
        let start = (10, 10);
        let goal = (11, 11);
        // Diagonal Distance: max(|10-11|, |10-11|) = max(1, 1) = 1
        let expected_heuristic: u16 = 1;

        let actual_heuristic = calculate_heuristic(start.0, start.1, goal.0, goal.1);

        assert_eq!(
            actual_heuristic, expected_heuristic,
            "Heuristic for adjacent diagonal cells should be 1 when using Diagonal Distance."
        );
    }

    #[test]
    fn test_heuristic_further_apart() {
        let start = (10, 10);
        let goal = (15, 18);
        // Diagonal Distance: max(|10-15|, |10-18|) = max(5, 8) = 8
        let expected_heuristic: u16 = 8;

        let actual_heuristic = calculate_heuristic(start.0, start.1, goal.0, goal.1);

        assert_eq!(
            actual_heuristic, expected_heuristic,
            "Heuristic for cells further apart should match the expected value for Diagonal Distance."
        );
    }

    #[test]
    fn test_heuristic_different_order() {
        // The heuristic should be the same regardless of the order of start and goal.
        let start1 = (10, 10);
        let goal1 = (15, 18);

        let start2 = (15, 18);
        let goal2 = (10, 10);

        let heuristic1 = calculate_heuristic(start1.0, start1.1, goal1.0, goal1.1);
        let heuristic2 = calculate_heuristic(start2.0, start2.1, goal2.0, goal2.1);

        assert_eq!(heuristic1, heuristic2, "Heuristic should be symmetrical.");
    }

    #[test]
    fn test_find_path_on_clear_board() {
        let board = Board::new(10, 10); // 10x10 clear board
        let start = (1, 1);
        let goal = (8, 8);

        // A direct diagonal path should be found
        let mut expected_path: VecDeque<(u16, u16)> = VecDeque::new();
        expected_path.push_back((2, 2));
        expected_path.push_back((3, 3));
        expected_path.push_back((4, 4));
        expected_path.push_back((5, 5));
        expected_path.push_back((6, 6));
        expected_path.push_back((7, 7));
        let expected_path = Some(expected_path);

        let actual_path = find_path_on_board(&board, start, goal);

        assert_eq!(
            actual_path, expected_path,
            "Should find a direct path on a clear board."
        );
    }

    #[test]
    fn test_find_path_around_single_obstacle_on_board() {
        let mut board = Board::new(10, 10); // 10x10 board

        // Place a wall obstacle
        board.change_base(BaseTerrain::Wall, 5, 5); // Obstacle at (5,5)

        let start = (4, 5);
        let goal = (6, 5);

        // Expected path should go around the obstacle (5,5).
        // Using Diagonal Distance heuristic and uniform cost 1, an optimal path has length 4 moves (5 nodes).
        // Examples: (4,5)->(5,4)->(6,5) OR (4,5)->(5,6)->(6,5) - this is longer
        let actual_path_option = find_path_on_board(&board, start, goal);

        assert!(
            actual_path_option.is_some(),
            "Should find a path around a single obstacle on the board."
        );
        let actual_path = actual_path_option.unwrap();
        println!("Path found around obstacle: {:?}", actual_path); // Print path to help debugging

        let end_path = actual_path.back().copied().unwrap();
        assert!(
            is_adjacent_to_goal(end_path, goal),
            "Path should end at the goal node."
        );
        assert!(
            !actual_path.contains(&(5, 5)),
            "Path should not include the obstacle cell."
        );

        // Check if the path length is optimal or close to optimal (for debugging, exact length is best)
        // For this specific case with Diagonal Distance heuristic and cost 1, the shortest path is 4 moves (5 nodes).
        assert_eq!(
            actual_path.len(),
            1,
            "Path around obstacle should have the optimal length (2 nodes)."
        );
    }

    #[test]
    fn test_find_path_around_impassable_entity_on_board() {
        let mut board = Board::new(10, 10); // 10x10 board

        // Place an impassable entity (like another Champion or Tower)
        board.place_cell(CellContent::Champion(99, Team::Red), 5, 5); // Entity at (5,5) - assuming Champions are impassable to minions

        let start = (4, 5);
        let goal = (6, 5);

        let actual_path_option = find_path_on_board(&board, start, goal);

        assert!(
            actual_path_option.is_some(),
            "Should find a path around an impassable entity on the board."
        );
        let actual_path = actual_path_option.unwrap();
        println!("Path found around entity: {:?}", actual_path); // Print path to help debugging

        let end_path = actual_path.back().copied().unwrap();
        assert!(
            is_adjacent_to_goal(end_path, goal),
            "Path should end at the goal node."
        );
        assert!(
            !actual_path.contains(&(5, 5)),
            "Path should not include the entity's cell."
        );

        // Check optimal path length (should be the same as around a wall in this scenario)
        assert_eq!(
            actual_path.len(),
            1,
            "Path around entity should have the optimal length (2 nodes)."
        );
    }

    #[test]
    fn test_find_path_no_path_exists_on_board() {
        let mut board = Board::new(10, 10); // 10x10 board

        // Surround the start cell with walls
        board.change_base(BaseTerrain::Wall, 4, 5);
        board.change_base(BaseTerrain::Wall, 5, 4);
        board.change_base(BaseTerrain::Wall, 5, 6);
        board.change_base(BaseTerrain::Wall, 6, 5);
        board.change_base(BaseTerrain::Wall, 6, 4);
        board.change_base(BaseTerrain::Wall, 4, 6);
        board.change_base(BaseTerrain::Wall, 4, 4);
        board.change_base(BaseTerrain::Wall, 6, 6);

        let start = (5, 5); // Start in the surrounded area
        let goal = (0, 0); // Goal far away

        let actual_path = find_path_on_board(&board, start, goal);

        assert!(
            actual_path.is_none(),
            "Should return None when no path exists on the board."
        );
    }

    #[test]
    fn test_find_path_around_wall_on_board() {
        let mut board = Board::new(10, 10); // 10x10 board

        // Create a vertical wall using BaseTerrain::Wall
        for r in 1..10 {
            board.change_base(BaseTerrain::Wall, r, 5); // Wall at col 5
        }

        let start = (5, 4); // To the left of the wall
        let goal = (5, 6); // To the right of the wall

        let actual_path_option = find_path_on_board(&board, start, goal);

        assert!(
            actual_path_option.is_some(),
            "Should find a path around a wall on the board."
        );
        let actual_path = actual_path_option.unwrap();
        println!("Path found around wall (Board): {:?}", actual_path);

        let end_path = actual_path.back().copied().unwrap();
        assert!(
            is_adjacent_to_goal(end_path, goal),
            "Path should end at the goal node."
        );
        for r in 1..10 {
            assert!(
                !actual_path.contains(&(r, 5)),
                "{}",
                &format!("Path should not include the wall cell ({}, 5).", r)
            );
        }

        // Check if the path length is optimal (11 nodes for 10 moves with Diagonal Distance cost 1)
        assert_eq!(
            actual_path.len(),
            9,
            "Path around wall should have the optimal length (4 nodes)."
        );
    }

    #[test]
    fn test_two_minions_pathfinding_to_same_goal() {
        let board = Board::new(10, 10); // A simple 10x10 board

        let minion1_start = (2, 5);
        let minion2_start = (1, 5);
        let goal = (6, 5);

        // Minion 1 pathfinding
        let path1_option = find_path_on_board(&board, minion1_start, goal);

        // Assert that a path was found for minion 1
        assert!(
            path1_option.is_some(),
            "Minion 1 should find a path to the goal's adjacent cell."
        );
        let path1 = path1_option.unwrap();

        // Assert that the path ends adjacent to the goal
        assert!(
            path1
                .back()
                .copied()
                .map_or(false, |pos| is_adjacent_to_goal(pos, goal)),
            "Minion 1 path should end at a cell adjacent to the goal."
        );

        println!("Minion 1 Path: {:?}", path1); // Print path for debugging

        // Minion 2 pathfinding
        let path2_option = find_path_on_board(&board, minion2_start, goal);

        // Assert that a path was found for minion 2
        assert!(
            path2_option.is_some(),
            "Minion 2 should find a path to the goal's adjacent cell."
        );
        let path2 = path2_option.unwrap();

        // Assert that the path ends adjacent to the goal
        assert!(
            path2
                .back()
                .copied()
                .map_or(false, |pos| is_adjacent_to_goal(pos, goal)),
            "Minion 2 path should end at a cell adjacent to the goal."
        );

        println!("Minion 2 Path: {:?}", path2); // Print path for debugging

        assert_eq!(
            path1.len(),
            3,
            "Minion 1 path should have the optimal length."
        );
        assert_eq!(
            path2.len(),
            4,
            "Minion 2 path should have the optimal length."
        );
    }

    #[test]
    fn test_eight_minions_dynamic_pathfinding_to_same_goal() {
        let mut board = Board::new(10, 10); // A simple 10x10 board

        // Define minions with their current position and an optional path
        struct TestMinion {
            id: u32, // Simple ID for tracking
            position: (u16, u16),
            path: Option<VecDeque<(u16, u16)>>,
            reached_goal_adjacent: bool,
        }

        let goal = (7, 6); // The common goal position

        // Starting positions for the 8 minions
        let minion_starts = vec![
            (0, 5), (1, 4), (1, 5), (1, 6), (2, 5), (3, 4), (3, 5), (3, 6)
        ];

        // Create the test minions and place them on the board initially
        let mut minions: Vec<TestMinion> = minion_starts.into_iter().enumerate().map(|(i, pos)| {
            // Place a dummy minion content on the board for pathfinding to see it as occupied
            board.place_cell(CellContent::Minion(i, Team::Blue), pos.0 as usize, pos.1 as usize);
            TestMinion {
                id: i as u32,
                position: pos,
                path: None,
                reached_goal_adjacent: false,
            }
        }).collect();

        let max_ticks = 50; // Prevent infinite loops in case of issues
        let mut all_reached = false;

        println!("--- Starting Dynamic Pathfinding Simulation ---");

        for tick in 0..max_ticks {
            println!("--- Tick {} ---", tick);
            all_reached = true; // Assume all will reach this tick

            // Shuffle minions to avoid favoring those processed first
            use rand::seq::SliceRandom;
            use rand::rng;
            minions.shuffle(&mut rng());


            for minion in &mut minions {
                if minion.reached_goal_adjacent {
                    continue; // This minion is done
                }

                all_reached = false; // At least one minion is not done

                // If the minion doesn't have a path, find one
                if minion.path.is_none() {
                    println!(" Minion {} at {:?} needs a path.", minion.id, minion.position);
                    minion.path = find_path_on_board(&board, minion.position, goal);

                    // If pathfinding failed, this minion is stuck for now
                    if minion.path.is_none() {
                        println!(" Minion {} at {:?} could not find a path.", minion.id, minion.position);
                        continue;
                    }
                }

                // If the minion has a path, attempt to move along it
                if let Some(path) = &mut minion.path {
                    if let Some(next_step) = path.front().copied() { // Peek at the next step
                         println!(" Minion {} at {:?} attempting to move to {:?}.", minion.id, minion.position, next_step);

                        // Check if the next step is passable *before* attempting the move
                        if board.get_cell(next_step.0 as usize, next_step.1 as usize).map_or(false, |cell| cell.is_passable()) {
                            // Attempt to move the minion: Clear old cell, update position, place on new cell
                            board.clear_cell(minion.position.0 as usize, minion.position.1 as usize);
                            minion.position = next_step;
                            // Place dummy minion content on the new cell
                             board.place_cell(CellContent::Minion(minion.id.try_into().unwrap(), Team::Blue), minion.position.0 as usize, minion.position.1 as usize);

                            path.pop_front(); // Remove the step from the path since move was successful
                             println!(" Minion {} successfully moved to {:?}. Path steps left: {}", minion.id, minion.position, path.len());


                            // Check if the minion has reached a cell adjacent to the goal after moving
                            if is_adjacent_to_goal(minion.position, goal) {
                                minion.reached_goal_adjacent = true;
                                minion.path = None; // Clear path once adjacent
                                println!(" Minion {} reached goal adjacent position {:?}.", minion.id, minion.position);
                            } else if path.is_empty() {
                                // Reached the end of a path but not adjacent to goal (shouldn't happen with correct A*)
                                println!(" Minion {} reached end of path but not adjacent to goal.", minion.id);
                                minion.path = None; // Clear path
                            }

                        } else {
                            // Next step in path is blocked (dynamic obstacle moved there)
                            println!(" Minion {}'s next step {:?} is blocked. Clearing path.", minion.id, next_step);
                            minion.path = None; // Clear path, will recalculate next tick
                        }
                    } else {
                        // Path was empty, clear it (should be handled by initial check, but good fallback)
                        minion.path = None;
                         println!(" Minion {}'s path was empty. Clearing path.", minion.id);
                    }
                }
            }

            // If all minions have reached goal adjacent, stop simulation
            if all_reached {
                println!("--- All minions reached goal adjacent positions. Simulation complete. ---");
                break;
            }

            // Optional: Add a delay here if you were visualizing
            // std::thread::sleep(std::time::Duration::from_millis(100));
        }


        // --- Assertions after simulation ---

        // Collect all final positions of the minions that reached goal adjacent
        let mut final_positions = HashSet::new();
        let mut reached_count = 0;

        for minion in &minions {
            if minion.reached_goal_adjacent {
                reached_count += 1;
                final_positions.insert(minion.position);

                // Assert that the final position is indeed adjacent to the goal
                 assert!(is_adjacent_to_goal(minion.position, goal),
                         "Minion {} final position {:?} should be adjacent to the goal {:?}.",
                         minion.id, minion.position, goal);
            }
        }

        println!("Minions reached goal adjacent: {}/{}", reached_count, minions.len());
        println!("Final adjacent positions: {:?}", final_positions);

        // Assert that all minions reached a goal adjacent cell
        assert_eq!(
            reached_count,
            minions.len(),
            "All minions should reach a cell adjacent to the goal."
        );

        // Assert that all final positions are unique (no two minions occupy the same cell)
        assert_eq!(
            final_positions.len(),
            minions.len(),
            "All minions should occupy unique cells adjacent to the goal."
        );

        // Optional: Assert that the set of final positions covers all 8 adjacent cells if 8 minions are used
        // and the goal is not near the board edge.
         if minions.len() == 8 {
             let expected_adjacent_cells: HashSet<(u16, u16)> = vec![
                 (goal.0 - 1, goal.1 - 1), (goal.0 - 1, goal.1), (goal.0 - 1, goal.1 + 1),
                 (goal.0, goal.1 - 1), /* (goal.0, goal.1), */ (goal.0, goal.1 + 1),
                 (goal.0 + 1, goal.1 - 1), (goal.0 + 1, goal.1), (goal.0 + 1, goal.1 + 1),
             ].into_iter().collect();

             // Filter expected cells to be within board bounds if necessary
             let bounded_expected_adjacent_cells: HashSet<(u16, u16)> = expected_adjacent_cells.into_iter()
                 .filter(|&(r, c)| usize::from(r) < board.rows && usize::from(c) < board.cols)
                 .collect();

             assert_eq!(
                 final_positions,
                 bounded_expected_adjacent_cells,
                 "The final positions should cover all valid adjacent cells around the goal."
             );
         }

    }
}

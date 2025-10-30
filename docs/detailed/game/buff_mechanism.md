# Buff and Debuff Mechanism

## Goal

The primary goal of the buff and debuff system is to introduce temporary status effects to game entities. This allows for more complex and strategic gameplay by enabling abilities that can alter the state of a champion or minion for a limited duration. The initial implementation focuses on a `Stun` effect, which prevents an entity from moving or attacking, but the system is designed to be extensible for other effects like slows, damage-over-time, or stat boosts.

## Core Components

The mechanism is centered around the `buffs` module and its integration into the game's entities and main game loop.

### The `buffs` Module (`game/src/game/buffs/`)

This module defines the core traits and concrete implementations for all status effects.

1.  **`Buff` Trait (`mod.rs`)**: This is the central trait that defines the behavior of any status effect.
    -   It requires `Send + Sync + Debug`, ensuring buffs are thread-safe and printable for debugging.
    -   **Key Methods**:
        -   `id()`: Returns a string slice to uniquely identify the buff (e.g., "Stun").
        -   `on_apply()`: Logic that executes once when the buff is first applied to a target.
        -   `on_tick()`: Logic that runs on every game tick. It returns a `bool` indicating if the buff has expired (`true`) or should continue (`false`).
        -   `on_remove()`: Logic that executes when the buff is removed, used for cleanup.

2.  **`HasBuff` Trait (`mod.rs`)**: This trait is implemented by any entity that can be affected by buffs (currently `Champion` and `Minion`). It provides a standardized way to query and alter the entity's state regarding common effects.
    -   **Key Methods**:
        -   `is_stunned()`: Checks if the entity is currently stunned.
        -   `set_stunned()`: Changes the entity's stunned state.

3.  **`StunBuff` Struct (`stun_buff.rs`)**: This is the concrete implementation for the stun effect.
    -   It holds the `duration_remaining` and the `applied_at` `Instant` to track its lifetime.
    -   `on_apply()` calls `target.set_stunned(true, ...)`.
    -   `on_tick()` checks if the elapsed time since application is greater than its duration.
    -   `on_remove()` calls `target.set_stunned(false, None)` to revert the effect.

4.  **`RedBuff` and `DoTBuff` Structs (`health_buff.rs`)**: These provide health-related effects.
    -   `RedBuff`: Grants bonus health regeneration per second for a duration.
    -   `DoTBuff`: Deals damage over time, applying damage per second to the target's health.

5.  **`BlueBuff` Struct (`mana_buff.rs`)**: Provides mana regeneration bonus.
    -   Grants bonus mana regeneration per second for a duration, similar to RedBuff but for mana.

6.  **Item Buffs (`item_buff.rs`)**: Permanent buffs from items.
    -   `HealthRegenItem`: Provides ongoing health regeneration.
    -   `ThornDamageItem`: Deals damage to attackers (passive effect).

### Entity Integration (`game/src/game/entities/`)

For an entity to be affected by buffs, it must be integrated into this system.

1.  **Storing Active Buffs**: Both `Champion` and `Minion` structs contain a field:
    ```rust
    pub active_buffs: HashMap<String, Box<dyn Buff>>
    ```
    This map holds all the status effects currently active on the entity, using the buff's ID as the key.

2.  **Implementing `HasBuff`**: Both `Champion` and `Minion` implement the `HasBuff` trait. They each have a `stun_timer: Option<Instant>` field. The `is_stunned` and `set_stunned` methods simply manage this timer to control the entity's state.

3.  **Receiving Buffs**: Buffs are applied when an entity's `Fighter::take_effect` method is called with a `Vec<GameplayEffect>`. This vector can contain multiple effects, including `GameplayEffect::Buff`.

## Interaction with the Game Loop (`GameManager::game_tick()`)

The lifecycle of every buff is managed centrally within the main game loop to ensure effects are updated and expire correctly.

1.  **Buff Application (Trigger)**:
    - A buff is typically applied as a result of another action. For example, when a projectile with a `Vec<GameplayEffect>` (which might include `GameplayEffect::Buff(...)`) hits a target, the `ProjectileManager` reports these effects.
    - The `GameManager` then calls the target's `take_effect` method, passing the `Vec<GameplayEffect>`, which processes each effect, including applying the `StunBuff` if present.

2.  **Buff Lifecycle Management (The "Tick")**:
    - This is the most critical part of the system and happens at the **very beginning** of each `game_tick`.
    - The manager iterates through every `Champion` and `Minion`.
    - To avoid Rust's borrow-checking conflicts, the process for each entity is:
        1.  **Take Buffs**: The entity's entire `active_buffs` `HashMap` is moved into a local variable using `std::mem::take`. The entity's own map is left empty.
        2.  **Process and Filter**: The system iterates through the now-local collection of buffs. For each buff:
            - It calls `buff.on_tick(entity)`.
            - If `on_tick` returns `true` (expired), `buff.on_remove(entity)` is called to clean up the effect, and the buff is discarded.
            - If `on_tick` returns `false` (still active), the buff is moved into a new, temporary `HashMap` of buffs to keep.
        3.  **Return Buffs**: The temporary map containing only the still-active buffs is moved back to become the entity's new `active_buffs`.

3.  **Enforcing Buff Effects**:
    - After the buff lifecycle is processed, the rest of the game tick proceeds.
    - When an entity attempts to perform an action, its internal logic checks its state. For example, `Minion::movement_phase()` and `Champion::take_action()` both check `self.is_stunned()` at the beginning. If the entity is stunned, the action is prevented.

This "take, filter, and replace" cycle ensures that buffs are managed safely and efficiently, providing a robust and extensible foundation for status effects in the game.

---

## How to Implement a New Buff

To add a new buff (e.g., a "Slow" effect that reduces movement speed), follow these steps:

1.  **Create the Buff Struct**:
    - In the `game/src/game/buffs/` directory, create a new file (e.g., `slow_buff.rs`).
    - Define a new struct (e.g., `SlowBuff`) and implement the `Buff` trait for it.

    ```rust
    // in slow_buff.rs
    use super::{Buff, HasBuff};
    use std::time::{Duration, Instant};

    #[derive(Debug)]
    pub struct SlowBuff {
        pub duration_remaining: Duration,
        pub applied_at: Instant,
    }

    impl Buff for SlowBuff {
        fn id(&self) -> &'static str { "Slow" }

        fn on_apply(&self, target: &mut dyn HasBuff) {
            target.set_slowed(true); // A new method you'll add to HasBuff
        }

        fn on_tick(&mut self, _target: &mut dyn HasBuff) -> bool {
            // Return true if expired
            self.applied_at.elapsed() >= self.duration_remaining
        }

        fn on_remove(&self, target: &mut dyn HasBuff) {
            target.set_slowed(false);
        }
    }
    ```

2.  **Update the `HasBuff` Trait**:
    - Open `game/src/game/buffs/mod.rs`.
    - Add methods to the `HasBuff` trait to manage the new effect's state.

    ```rust
    // in buffs/mod.rs
    pub trait HasBuff: Send + Sync {
        // ... existing methods ...
        fn is_slowed(&self) -> bool;
        fn set_slowed(&mut self, is_slowed: bool);
    }
    ```

3.  **Implement the Trait for Entities**:
    - Open `game/src/game/entities/champion.rs` and `minion.rs`.
    - Add a new field to track the slowed state (e.g., `is_slowed: bool`).
    - Implement the new `is_slowed` and `set_slowed` methods for both `Champion` and `Minion`.

    ```rust
    // Example for Champion
    impl HasBuff for Champion {
        // ...
        fn is_slowed(&self) -> bool { self.is_slowed }
        fn set_slowed(&mut self, is_slowed: bool) { self.is_slowed = is_slowed; }
    }
    ```

4.  **Enforce the Effect**:
    - In the game logic, check for the new state where it matters. For a slow, you would modify the movement logic.

    ```rust
    // Example in Champion::move_in_direction
    fn move_in_direction(&mut self, direction: Direction) {
        if self.is_stunned() { return; } // Existing check

        let move_speed = if self.is_slowed() { 2.0 } else { 1.0 }; // Apply slow effect
        // ... apply movement with modified speed ...
    }
    ```

5.  **Add the Buff to the `GameplayEffect` Enum**:
    - Finally, update `game/src/game/gameplay_effect.rs` to allow the new buff to be passed as an effect.

    ```rust
    // in gameplay_effect.rs
    pub enum GameplayEffect {
        Damage(u16),
        Buff(BuffType),
        // ...
    }

    pub enum BuffType {
        Stun,
        Slow, // Add new buff type
    }
    ```
    - You will also need to update the logic that creates the `Box<dyn Buff>` from the `BuffType` enum, likely where `GameplayEffect::Buff` is processed.


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

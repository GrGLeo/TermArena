# Spell Selection Logic (Future Implementation)

This document outlines the high-level logic for implementing spell selection in the game, leveraging Rust's trait system for extensibility.

## Goal
Allow players to select 2 spells in the lobby, which will then be available for use during gameplay via generic action buttons (e.g., Action1, Action2).

## Core Concepts
- **Traits for Polymorphism:** Define a `Spell` trait that all concrete spell implementations must adhere to. This allows treating different spells uniformly.
- **Trait Objects:** Store instances of different spell types as `Box<dyn Spell>` (trait objects) in the `Champion` struct, enabling dynamic dispatch.

## Implementation Steps

### 1. Define the `Spell` Trait
- Create a `Spell` trait (e.g., in `game/src/game/spell/spell_trait.rs`).
- It should define common methods like `cast(...)`, `name()`, and `get_stats()`.
- The `cast` method will encapsulate the spell's unique effects, taking `caster`, `board`, and `projectile_manager` as arguments.

### 2. Implement Concrete Spell Structs
- For each spell (e.g., `FreezeWallSpell`, `FireballSpell`), create a dedicated struct.
- Each struct will hold its specific `SpellStats` (loaded from `stats.toml`).
- Implement the `Spell` trait for each concrete spell struct, providing the actual logic for its `cast` method.
- The existing `cast_freeze_wall` function's logic will be moved into the `impl Spell for FreezeWallSpell` block.

### 3. Update `Champion` Struct
- Modify `Champion` to store a `HashMap<String, Box<dyn Spell>>` (e.g., `selected_spells`) to hold the actual spell implementations.
- Add `Option<String>` fields (e.g., `selected_spell_slot_1`, `selected_spell_slot_2`) to map generic action buttons to the names of the selected spells.

### 4. Update `Champion::new`
- The constructor will now accept the `HashMap<String, Box<dyn Spell>>` and the `Option<String>` for the selected spell slots.
- It will populate the `Champion`'s new fields accordingly.

### 5. Update `Champion::take_action`
- When `Action::Action1` or `Action::Action2` is triggered:
    - Retrieve the spell's name from `selected_spell_slot_1` or `selected_spell_slot_2`.
    - Use this name to look up the corresponding `Box<dyn Spell>` from `selected_spells`.
    - Call the generic `cast` method on the retrieved trait object. This method will execute the specific spell's logic without `take_action` needing to know its concrete type.

### 6. Update `GameManager::add_player`
- When a player joins and selects spells:
    - Retrieve the `SpellStats` for the selected spells from `GameConfig.spells`.
    - Instantiate the concrete spell structs (e.g., `FreezeWallSpell::new(stats)`), box them, and collect them into the `HashMap<String, Box<dyn Spell>>`.
    - Pass this map, along with the selected spell names, to the `Champion::new` constructor.

## Benefits
- **Extensibility:** Easily add new spells by creating new structs and implementing the `Spell` trait, without modifying existing `Champion` or `GameManager` logic.
- **Modularity:** Spell logic is encapsulated within its own struct.
- **Maintainability:** Centralized management of spell behavior through the `Spell` trait.
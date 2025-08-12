# Casting Mechanism

The casting mechanism in the game allows for abilities and spells to have a cast time, during which the champion is busy performing the action. This document explains how the system works and how to add new castable actions.

## Core Components

The casting system is primarily composed of three components defined in `game/src/game/entities/champion.rs`:

-   **`Cast` struct**: This struct represents an active cast and holds all the necessary information.
    -   `start_time`: An `Instant` marking when the cast began.
    -   `duration`: A `Duration` specifying how long the cast takes to complete.
    -   `action`: A `Castable` enum that defines what action to perform when the cast is complete.

-   **`Castable` enum**: This enum represents any action that can be cast over time.
    -   `Spell(Box<dyn Spell>)`: A spell that has a cast time.
    -   `Ability(Ability)`: A fundamental champion ability, like `Recall`.

-   **`Ability` enum**: This enum defines the basic, non-spell abilities that a champion can cast.
    -   `Recall`: Teleports the champion back to the base after a cast time.

## How it Works

1.  **Initiating a Cast**: When a player triggers a castable action (e.g., `Action::Recall`), a new `Cast` struct is created and stored in the champion's `current_cast` field.

2.  **Game Loop Check**: In the main `game_tick` loop (`game/src/game/mod.rs`), the system checks each champion for an active cast.

3.  **Cast Completion**: If a cast is active, the system checks if `cast.start_time.elapsed() >= cast.duration`. If the cast is complete, the corresponding action from `cast.action` is executed.

4.  **Cast Interruption**: A cast can be interrupted. Currently, any movement action will interrupt an ongoing cast. When a champion moves, the `current_cast` is set to `None`, effectively cancelling the action.

## Implementing a New Castable Ability

To add a new castable ability (e.g., a "Channeling Shield"):

1.  **Add to `Ability` enum**: Add a new variant to the `Ability` enum in `champion.rs`.
    ```rust
    pub enum Ability {
        Recall,
        ChannelingShield, // New ability
    }
    ```

2.  **Add to `Action` enum**: Add a corresponding action to the `Action` enum in `champion.rs`.
    ```rust
    pub enum Action {
        // ...
        Recall,
        ChannelShield, // New action
        // ...
    }
    ```

3.  **Handle the Action**: In `Champion::take_action`, add a new arm to the `match` statement to handle the new action and create a `Cast`.
    ```rust
    Action::ChannelShield => {
        self.current_cast = Some(Cast {
            start_time: Instant::now(),
            duration: Duration::from_secs(3),
            action: Castable::Ability(Ability::ChannelingShield),
        });
        Ok(())
    }
    ```

4.  **Implement the Effect**: In `game_tick` in `game/mod.rs`, add the logic to execute the ability's effect when the cast completes.
    ```rust
    // in game_tick, inside the cast completion logic
    match &cast.action {
        Castable::Ability(Ability::Recall) => { ... }
        Castable::Ability(Ability::ChannelingShield) => {
            // Apply a shield buff to the champion
        }
        Castable::Spell(_) => { ... }
    }
    ```

## Implementing a Castable Spell

To make a spell have a cast time:

1.  **Add `cast_time` to Spell Stats**: You would typically add a `cast_time` property to your spell's configuration (e.g., in `spells.toml`).

2.  **Modify Spell Casting in `take_action`**: When the spell is cast in `Champion::take_action`, instead of executing its effect immediately, create a `Cast` instance.
    ```rust
    // Example for a spell action
    Action::Action3 => { // A new spell
        if let Some(spell) = self.spells.get(&SPELL_ID) {
            if spell.cast_time() > 0 {
                self.current_cast = Some(Cast {
                    start_time: Instant::now(),
                    duration: Duration::from_millis(spell.cast_time()),
                    action: Castable::Spell(spell.clone_box()),
                });
            } else {
                // Instant cast
                spell.cast(self, ...);
            }
        }
        Ok(())
    }
    ```

3.  **Handle Spell Completion in `game_tick`**: In the `game_tick`'s cast completion logic, you'll need to execute the spell's effect.
    ```rust
    // in game_tick
    match &cast.action {
        Castable::Ability(...) => { ... }
        Castable::Spell(spell) => {
            spell.cast(champ, ...);
        }
    }
    ```

# Game Mechanics Overview

This directory contains detailed documentation for the core real-time gameplay mechanics handled by the Rust game server. These systems work together to create a dynamic and interactive experience.

## Core Systems

- **[Buff and Debuff Mechanism](./buff_mechanism.md)**
  - This system manages temporary status effects on game entities (Champions, Minions). It is responsible for applying, ticking, and removing effects like stuns, slows, or damage-over-time. It is a flexible system designed to be easily extended with new effects.

- **[Casting Mechanism](./cast_mechanism.md)**
  - This system handles actions that take time to complete, such as casting a powerful spell or recalling to base. It ensures that players are occupied during the cast and that the action can be interrupted by movement, adding a layer of risk and strategy.

- **[Projectile Mechanism](./projectile_mechanism.md)**
  - This system governs all non-instant attacks and spells. It is responsible for creating, moving, and detecting collisions for projectiles. This allows for skillshots that can be dodged by opponents, a key feature of tactical gameplay.

## How They Interact

These systems are deeply interconnected within the main game loop:

1.  A player action, handled by the **Casting Mechanism**, might initiate a spell.
2.  Once the cast is complete, the spell might spawn a projectile, which is now managed by the **Projectile Mechanism**.
3.  When the projectile hits a target, the **Projectile Mechanism** applies a `GameplayEffect`.
4.  This effect could be a buff (e.g., a stun), which is then applied to the target entity and managed by the **Buff Mechanism**.

Understanding these three systems is key to developing new gameplay features. Each document provides a detailed explanation of its own components and a guide for how to extend it.

// TTK Calculator for Term Arena
class TTKCalculator {
    constructor() {
        this.stats = {};
        this.items = [];
        this.equippedItems = [];
        this.currentStats = {};
        this.neutralMonsters = [];
        this.manualModifications = {}; // Store manual stat modifications per entity: {entityName: {stat: value}}

        this.init();
    }

    async init() {
        await this.loadConfigFiles();
        this.buildEntityUI();
        this.buildItemUI();
        this.setupEventListeners();
        this.calculateAndDisplayTTK();
    }

    async loadConfigFiles() {
        try {
            let loadErrors = [];

            // Load stats.toml
            try {
                const timestamp = Date.now();
                const statsResponse = await fetch(`../../services/game/stats.toml?t=${timestamp}`);
                if (!statsResponse.ok) {
                    throw new Error(`HTTP ${statsResponse.status}: ${statsResponse.statusText}`);
                }
                const statsText = await statsResponse.text();
                this.stats = TOML.parse(statsText);
            } catch (error) {
                loadErrors.push(`stats.toml: ${error.message}`);
                console.error('Error loading stats.toml:', error);
            }

            // Load items.toml
            try {
                const timestamp = Date.now();
                const itemsResponse = await fetch(`../../services/game/items.toml?t=${timestamp}`);
                if (!itemsResponse.ok) {
                    throw new Error(`HTTP ${itemsResponse.status}: ${itemsResponse.statusText}`);
                }
                const itemsText = await itemsResponse.text();
                const itemsData = TOML.parse(itemsText);
                this.items = itemsData.items || [];
            } catch (error) {
                loadErrors.push(`items.toml: ${error.message}`);
                console.error('Error loading items.toml:', error);
                this.items = [];
            }



            // Extract neutral monsters (only if stats loaded successfully)
            if (this.stats) {
                this.neutralMonsters = this.stats.neutral_monsters || [];
            } else {
                this.neutralMonsters = [];
            }

            // Initialize current stats as copies (only if stats loaded)
            if (this.stats) {
                this.currentStats = JSON.parse(JSON.stringify(this.stats));
                // Clear manual modifications on initialization
                this.manualModifications = {};
                // Initialize manual modifications for all entities
                Object.keys(this.stats).forEach(entityName => {
                    this.manualModifications[entityName] = {};
                });
                this.neutralMonsters.forEach(monster => {
                    this.manualModifications[monster.id] = {};
                });
            }

            if (loadErrors.length > 0) {
                alert('Some config files failed to load:\n' + loadErrors.join('\n'));
            }
        } catch (error) {
            console.error('Unexpected error loading config files:', error);
            alert('Unexpected error loading config files: ' + error.message);
        }
    }

    buildEntityUI() {
        const entityList = document.getElementById('entity-list');
        entityList.innerHTML = '';

        // Get selected entities from TTK Results
        const attackerSelect = document.getElementById('attacker-select');
        const defenderSelect = document.getElementById('defender-select');
        const selectedEntities = new Set([
            attackerSelect.value,
            defenderSelect.value
        ]);

        // Build UI for selected entities
        selectedEntities.forEach(entityName => {
            let entityStats;

            if (entityName === 'champion') {
                // Use current stats for champion (includes manual modifications + item bonuses)
                entityStats = this.currentStats.champion;
            } else if (entityName === 'minion') {
                entityStats = JSON.parse(JSON.stringify(this.stats.minion || {}));
            } else if (entityName === 'tower') {
                entityStats = JSON.parse(JSON.stringify(this.stats.tower || {}));
            } else {
                // Check if it's a neutral monster
                const monster = this.neutralMonsters.find(m => m.id === entityName);
                if (monster) {
                    entityStats = JSON.parse(JSON.stringify(monster));
                }
            }

            // Apply manual modifications for this entity
            if (entityStats && this.manualModifications[entityName]) {
                Object.entries(this.manualModifications[entityName]).forEach(([stat, value]) => {
                    if (entityStats[stat] !== undefined) {
                        entityStats[stat] = value;
                    }
                });
            }

            if (entityStats) {
                const entityDiv = this.createEntityControls(entityName, entityStats);
                entityList.appendChild(entityDiv);
                // Update the UI values for this entity
                this.updateEntityUIValues(entityName);
            }
        });
    }

    createEntityControls(entityName, entityStats) {
        const entityDiv = document.createElement('div');
        entityDiv.className = 'entity-controls';
        const attackSpeedControl = entityStats.attack_speed_ms ?
            this.createStatControl(entityName, 'attack_speed_ms', entityStats.attack_speed_ms) :
            this.createStatControl(entityName, 'attack_speed_secs', entityStats.attack_speed_secs || 0);

        entityDiv.innerHTML = `
            <h3>${entityName.charAt(0).toUpperCase() + entityName.slice(1)}</h3>
            <div class="stat-controls">
                ${this.createStatControl(entityName, 'health', entityStats.health || 0)}
                ${this.createStatControl(entityName, 'attack_damage', entityStats.attack_damage || 0)}
                ${this.createStatControl(entityName, 'armor', entityStats.armor || 0)}
                ${attackSpeedControl}
            </div>
        `;

        return entityDiv;
    }

    createStatControl(entityName, statName, initialValue) {
        const displayName = statName.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase());
        return `
            <div class="stat-control">
                <label>${displayName}:</label>
                <button class="btn-decrease" data-entity="${entityName}" data-stat="${statName}">-</button>
                <input type="number" class="stat-value" data-entity="${entityName}" data-stat="${statName}" value="${initialValue}" min="0" step="1">
                <button class="btn-increase" data-entity="${entityName}" data-stat="${statName}">+</button>
            </div>
        `;
    }

    buildItemUI() {
        const availableItems = document.getElementById('available-items');
        availableItems.innerHTML = '<h4>Available Items</h4>';

        this.items.forEach(item => {
            const itemDiv = document.createElement('div');
            itemDiv.className = 'item-card';
            itemDiv.innerHTML = `
                <h5>${item.name}</h5>
                <p>Cost: ${item.cost}</p>
                <div class="item-stats">
                    ${this.formatItemStats(item.stats)}
                </div>
                <button class="btn-equip" data-item-id="${item.id}">Equip</button>
            `;
            availableItems.appendChild(itemDiv);
        });

        this.updateEquippedItemsUI();
    }

    formatItemStats(stats) {
        if (!stats) return '';
        return Object.entries(stats)
            .map(([key, value]) => `${key.replace(/_/g, ' ')}: +${value}`)
            .join('<br>');
    }



    setupEventListeners() {
        // Stat modification buttons
        document.addEventListener('click', (e) => {
            if (e.target.classList.contains('btn-increase') || e.target.classList.contains('btn-decrease')) {
                this.modifyStat(e.target);
            }
        });

        // Stat input changes
        document.addEventListener('input', (e) => {
            if (e.target.classList.contains('stat-value')) {
                this.updateStatFromInput(e.target);
            }
        });

        // Item equip buttons
        document.addEventListener('click', (e) => {
            if (e.target.classList.contains('btn-equip')) {
                this.equipItem(parseInt(e.target.dataset.itemId));
            }
        });

        // Item unequip buttons
        document.addEventListener('click', (e) => {
            if (e.target.classList.contains('btn-unequip')) {
                this.unequipItem(parseInt(e.target.dataset.itemId));
            }
        });

        // Attacker/Defender selection changes
        document.getElementById('attacker-select').addEventListener('change', () => {
            this.buildEntityUI();
            this.calculateAndDisplayTTK();
        });
        document.getElementById('defender-select').addEventListener('change', () => {
            this.buildEntityUI();
            this.calculateAndDisplayTTK();
        });
    }

    modifyStat(button) {
        const entity = button.dataset.entity;
        const stat = button.dataset.stat;
        const input = document.querySelector(`input[data-entity="${entity}"][data-stat="${stat}"]`);
        let currentValue = parseFloat(input.value) || 0;

        if (button.classList.contains('btn-increase')) {
            currentValue += 1;
        } else {
            currentValue = Math.max(0, currentValue - 1);
        }

        input.value = currentValue;

        if (entity === 'champion') {
            // Store manual modification
            if (!this.manualModifications[entity]) {
                this.manualModifications[entity] = {};
            }
            const baseValue = this.stats.champion[stat];
            if (currentValue !== baseValue) {
                this.manualModifications[entity][stat] = currentValue;
            } else {
                delete this.manualModifications[entity][stat];
            }
            // Recalculate stats to preserve item bonuses
            this.recalculateChampionStats();
        } else {
            // Store manual modification for other entities
            if (!this.manualModifications[entity]) {
                this.manualModifications[entity] = {};
            }
            const baseStats = this.stats[entity] || this.neutralMonsters.find(m => m.id === entity);
            if (baseStats && baseStats[stat] !== undefined) {
                const baseValue = baseStats[stat];
                if (currentValue !== baseValue) {
                    this.manualModifications[entity][stat] = currentValue;
                } else {
                    delete this.manualModifications[entity][stat];
                }
            }
            this.updateCurrentStats(entity, stat, currentValue);
        }
        this.calculateAndDisplayTTK();
    }

    updateStatFromInput(input) {
        const entity = input.dataset.entity;
        const stat = input.dataset.stat;
        const value = parseFloat(input.value) || 0;

        if (entity === 'champion') {
            // Store manual modification
            if (!this.manualModifications[entity]) {
                this.manualModifications[entity] = {};
            }
            const baseValue = this.stats.champion[stat];
            if (value !== baseValue) {
                this.manualModifications[entity][stat] = value;
            } else {
                delete this.manualModifications[entity][stat];
            }
            // Recalculate stats to preserve item bonuses
            this.recalculateChampionStats();
        } else {
            // Store manual modification for other entities
            if (!this.manualModifications[entity]) {
                this.manualModifications[entity] = {};
            }
            const baseStats = this.stats[entity] || this.neutralMonsters.find(m => m.id === entity);
            if (baseStats && baseStats[stat] !== undefined) {
                const baseValue = baseStats[stat];
                if (value !== baseValue) {
                    this.manualModifications[entity][stat] = value;
                } else {
                    delete this.manualModifications[entity][stat];
                }
            }
            this.updateCurrentStats(entity, stat, value);
        }
        this.calculateAndDisplayTTK();
    }

    updateCurrentStats(entity, stat, value) {
        if (!this.currentStats[entity]) {
            // Find in neutral monsters
            const monster = this.neutralMonsters.find(m => m.id === entity);
            if (monster) {
                monster[stat] = value;
            }
        } else {
            this.currentStats[entity][stat] = value;
        }
    }

    equipItem(itemId) {
        if (this.equippedItems.length >= 6) {
            alert('Maximum 6 items allowed!');
            return;
        }

        const item = this.items.find(i => i.id === itemId);
        if (!item) return;

        // Allow multiple instances of the same item
        this.equippedItems.push(item);
        this.updateEquippedItemsUI();
        this.recalculateChampionStats();
        this.calculateAndDisplayTTK();
    }

    unequipItem(itemId) {
        this.equippedItems = this.equippedItems.filter(item => item.id !== itemId);
        this.updateEquippedItemsUI();
        this.recalculateChampionStats();
        this.calculateAndDisplayTTK();
    }

    updateEquippedItemsUI() {
        const equippedContainer = document.getElementById('equipped-items');
        equippedContainer.innerHTML = '<h4>Equipped Items</h4>';

        this.equippedItems.forEach(item => {
            const itemDiv = document.createElement('div');
            itemDiv.className = 'equipped-item';
            itemDiv.innerHTML = `
                <h5>${item.name}</h5>
                <div class="item-stats">
                    ${this.formatItemStats(item.stats)}
                </div>
                <button class="btn-unequip" data-item-id="${item.id}">Unequip</button>
            `;
            equippedContainer.appendChild(itemDiv);
        });
    }

    recalculateChampionStats() {
        // Calculate what the stats should be: base + manual modifications + item bonuses
        const baseChampion = this.stats.champion;
        const resultStats = JSON.parse(JSON.stringify(baseChampion));

        // Apply stored manual modifications for champion
        if (this.manualModifications.champion) {
            Object.entries(this.manualModifications.champion).forEach(([stat, value]) => {
                if (resultStats[stat] !== undefined) {
                    resultStats[stat] = value;
                }
            });
        }

        // Apply item bonuses on top
        this.equippedItems.forEach(item => {
            if (item.stats) {
                Object.entries(item.stats).forEach(([stat, value]) => {
                    if (resultStats[stat] !== undefined) {
                        resultStats[stat] += value;
                    }
                });
            }
        });

        // Update current stats
        this.currentStats.champion = resultStats;

        // Update UI to reflect new stats
        this.updateEntityUIValues('champion');
    }

    updateEntityUIValues(entityName) {
        let entityStats;

        if (entityName === 'champion') {
            entityStats = this.currentStats.champion;
        } else {
            // For other entities, get base stats and apply manual modifications
            if (this.stats[entityName]) {
                entityStats = JSON.parse(JSON.stringify(this.stats[entityName]));
            } else {
                const monster = this.neutralMonsters.find(m => m.id === entityName);
                if (monster) {
                    entityStats = JSON.parse(JSON.stringify(monster));
                }
            }

            // Apply manual modifications
            if (entityStats && this.manualModifications[entityName]) {
                Object.entries(this.manualModifications[entityName]).forEach(([stat, value]) => {
                    if (entityStats[stat] !== undefined) {
                        entityStats[stat] = value;
                    }
                });
            }
        }

        if (!entityStats) return;

        Object.entries(entityStats).forEach(([stat, value]) => {
            const input = document.querySelector(`input[data-entity="${entityName}"][data-stat="${stat}"]`);
            if (input) {
                input.value = value;
            }
        });
    }

    calculateAndDisplayTTK() {
        const attackerSelect = document.getElementById('attacker-select');
        const defenderSelect = document.getElementById('defender-select');

        // Store current selection before updating options
        const currentAttacker = attackerSelect.value;
        const currentDefender = defenderSelect.value;

        // Clear existing options and add current entities
        this.updateEntitySelectors();

        // Restore the selection if the options still exist
        if (Array.from(attackerSelect.options).some(opt => opt.value === currentAttacker)) {
            attackerSelect.value = currentAttacker;
        }
        if (Array.from(defenderSelect.options).some(opt => opt.value === currentDefender)) {
            defenderSelect.value = currentDefender;
        }

        const attackerName = attackerSelect.value;
        const defenderName = defenderSelect.value;

        if (!attackerName || !defenderName) return;

        const attackerStats = this.getEntityStats(attackerName);
        const defenderStats = this.getEntityStats(defenderName);

        if (!attackerStats || !defenderStats) return;

        const result = this.calculateTTK(attackerStats, defenderStats, attackerName === 'champion');

        this.displayTTKResult(attackerName, defenderName, result);
    }

    updateEntitySelectors() {
        const attackerSelect = document.getElementById('attacker-select');
        const defenderSelect = document.getElementById('defender-select');

        // Clear existing neutral monster options (keep the original 3: champion, minion, tower)
        const originalOptions = ['champion', 'minion', 'tower'];
        const attackerOptions = Array.from(attackerSelect.options);
        const defenderOptions = Array.from(defenderSelect.options);

        // Remove any options that aren't the original 3
        attackerOptions.forEach(option => {
            if (!originalOptions.includes(option.value)) {
                attackerSelect.removeChild(option);
            }
        });
        defenderOptions.forEach(option => {
            if (!originalOptions.includes(option.value)) {
                defenderSelect.removeChild(option);
            }
        });

        // Add neutral monsters
        this.neutralMonsters.forEach(monster => {
            const option1 = new Option(monster.id, monster.id);
            const option2 = new Option(monster.id, monster.id);
            attackerSelect.appendChild(option1);
            defenderSelect.appendChild(option2);
        });
    }

    getEntityStats(entityName) {
        if (this.currentStats[entityName]) {
            return this.currentStats[entityName];
        }

        // Check neutral monsters
        return this.neutralMonsters.find(m => m.id === entityName);
    }

    calculateTTK(attacker, defender, isChampionAttacker = false) {
        // Calculate attacks per second
        let attacksPerSecond;
        if (attacker.attack_speed_ms) {
            attacksPerSecond = 1000 / attacker.attack_speed_ms;
        } else if (attacker.attack_speed_secs) {
            attacksPerSecond = 1 / attacker.attack_speed_secs;
        } else {
            attacksPerSecond = 1; // fallback
        }

        // Calculate damage per hit
        const damagePerHit = attacker.attack_damage * (100 / (100 + defender.armor));

        // Calculate hits to kill
        let hitsToKill;
        if (damagePerHit > 0) {
            hitsToKill = Math.ceil(defender.health / damagePerHit);
        } else {
            hitsToKill = Infinity;
        }

        // Calculate time to kill
        let timeToKill;
        if (attacksPerSecond > 0) {
            timeToKill = hitsToKill / attacksPerSecond;
        } else {
            timeToKill = Infinity;
        }

        // Calculate DPS
        const dps = damagePerHit * attacksPerSecond;

        // Calculate defender's attack speed
        let defenderAttacksPerSecond;
        if (defender.attack_speed_ms) {
            defenderAttacksPerSecond = 1000 / defender.attack_speed_ms;
        } else if (defender.attack_speed_secs) {
            defenderAttacksPerSecond = 1 / defender.attack_speed_secs;
        } else {
            defenderAttacksPerSecond = 1; // fallback
        }

        // Calculate defender's damage per hit to attacker
        const defenderDamagePerHit = defender.attack_damage * (100 / (100 + attacker.armor));

        // Calculate how many attacks defender can make during the fight
        const defenderTotalAttacks = Math.floor(timeToKill * defenderAttacksPerSecond);

        // Calculate total damage dealt to attacker
        const totalDamageToAttacker = defenderTotalAttacks * defenderDamagePerHit;

        // Calculate attacker's remaining HP after the fight
        const attackerRemainingHP = attacker.health - totalDamageToAttacker;

        return {
            hitsToKill: hitsToKill,
            timeToKill: timeToKill,
            dps: dps,
            attackerRemainingHP: attackerRemainingHP
        };
    }

    displayTTKResult(attackerName, defenderName, result) {
        const tbody = document.querySelector('#ttk-table tbody');
        tbody.innerHTML = '';

        const row = document.createElement('tr');
        const attackerHPLeft = result.attackerRemainingHP <= 0 ?
            `<span style="color: red;">${result.attackerRemainingHP.toFixed(0)}</span>` :
            `<span style="color: green;">${result.attackerRemainingHP.toFixed(0)}</span>`;

        row.innerHTML = `
            <td>${attackerName}</td>
            <td>${defenderName}</td>
            <td>${result.hitsToKill === Infinity ? '∞' : result.hitsToKill}</td>
            <td>${result.timeToKill === Infinity ? '∞' : result.timeToKill.toFixed(2)}</td>
            <td>${result.dps.toFixed(2)}</td>
            <td>${attackerHPLeft}</td>
        `;
        tbody.appendChild(row);
    }
}

// Initialize the calculator when the page loads
document.addEventListener('DOMContentLoaded', () => {
    new TTKCalculator();
});

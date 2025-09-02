// Test script to verify TOML parsing
async function testTOMLParsing() {
    try {
        console.log('Testing TOML parsing...');

        // Test items.toml
        const itemsResponse = await fetch('../../services/game/items.toml');
        const itemsText = await itemsResponse.text();
        const itemsData = TOML.parse(itemsText);
        console.log('Items data:', itemsData);
        console.log('Items array:', itemsData.items);

        // Test spells.toml
        const spellsResponse = await fetch('../../services/game/spells.toml');
        const spellsText = await spellsResponse.text();
        const spellsData = TOML.parse(spellsText);
        console.log('Spells data:', spellsData);
        console.log('Spells array:', spellsData.spell);

        // Test stats.toml
        const statsResponse = await fetch('../../services/game/stats.toml');
        const statsText = await statsResponse.text();
        const statsData = TOML.parse(statsText);
        console.log('Stats data:', statsData);
        console.log('Neutral monsters:', statsData.neutral_monsters);

    } catch (error) {
        console.error('TOML parsing test failed:', error);
    }
}

// Run the test
testTOMLParsing();
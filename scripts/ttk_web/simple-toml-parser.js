// Simple TOML Parser for basic config files
function parseTOML(tomlText) {
    const result = {};
    const lines = tomlText.split('\n').map(line => line.trim()).filter(line => line && !line.startsWith('#'));

    let currentSection = result;
    let currentArray = null;
    let arrayKey = null;

    for (let i = 0; i < lines.length; i++) {
        const line = lines[i];

        // Section header
        if (line.startsWith('[') && line.endsWith(']') && !line.startsWith('[[')) {
            const sectionName = line.slice(1, -1);
            if (!result[sectionName]) {
                result[sectionName] = {};
            }
            currentSection = result[sectionName];
            currentArray = null;
            continue;
        }

        // Array of tables
        if (line.startsWith('[[') && line.endsWith(']]')) {
            const arrayName = line.slice(2, -2).trim();
            if (!result[arrayName]) {
                result[arrayName] = [];
            }
            currentArray = result[arrayName];
            const newItem = {};
            currentArray.push(newItem);
            currentSection = newItem;
            continue;
        }

        // Key-value pair
        if (line.includes('=')) {
            const [key, ...valueParts] = line.split('=');
            const keyName = key.trim();
            let valueStr = valueParts.join('=').trim();

            // Remove quotes if present
            if ((valueStr.startsWith('"') && valueStr.endsWith('"')) ||
                (valueStr.startsWith("'") && valueStr.endsWith("'"))) {
                valueStr = valueStr.slice(1, -1);
            }

            // Parse value
            let value;
            if (valueStr === 'true') {
                value = true;
            } else if (valueStr === 'false') {
                value = false;
            } else if (/^\d+$/.test(valueStr)) {
                value = parseInt(valueStr);
            } else if (/^\d+\.\d+$/.test(valueStr)) {
                value = parseFloat(valueStr);
            } else if (valueStr.startsWith('[') && valueStr.endsWith(']')) {
                // Array
                const arrayContent = valueStr.slice(1, -1);
                value = arrayContent.split(',').map(item => {
                    const trimmed = item.trim();
                    if (trimmed === 'true') return true;
                    if (trimmed === 'false') return false;
                    if (/^\d+$/.test(trimmed)) return parseInt(trimmed);
                    if (/^\d+\.\d+$/.test(trimmed)) return parseFloat(trimmed);
                    return trimmed.replace(/"/g, '');
                });
            } else {
                value = valueStr;
            }

            // Handle nested keys (e.g., stats.attack_damage)
            const keyParts = keyName.split('.');
            let targetSection = currentSection;

            // Create nested objects for dotted keys
            for (let i = 0; i < keyParts.length - 1; i++) {
                const part = keyParts[i];
                if (!targetSection[part]) {
                    targetSection[part] = {};
                }
                targetSection = targetSection[part];
            }

            targetSection[keyParts[keyParts.length - 1]] = value;
        }
    }

    return result;
}

// Make it available globally
window.TOML = { parse: parseTOML };

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

// Read zh.ts and extract all keys using regex or ast.
// We'll just do a rough extraction
const zhContent = fs.readFileSync('d:/Go/src/cftpweb/candidateserver/web/lib/locales/zh.ts', 'utf8');

const keys = [];
let currentObj = null;

const lines = zhContent.split('\n');
for (let line of lines) {
  const objMatch = line.match(/^  ([a-zA-Z0-9_]+): \{/);
  if (objMatch) {
    currentObj = objMatch[1];
    continue;
  }
  
  if (line.match(/^  \},/)) {
    currentObj = null;
    continue;
  }
  
  if (currentObj) {
    const keyMatch = line.match(/^\s+([a-zA-Z0-9_]+):/);
    if (keyMatch) {
      keys.push(`t.${currentObj}.${keyMatch[1]}`);
    }
  }
}

console.log(`Found ${keys.length} keys`);
const unused = [];

// Grep all keys
for (let key of keys) {
  try {
    // using grep to find key
    // escape dots
    const escapedKey = key.replace(/\./g, '\\.');
    const cmd = `findstr /s /m /c:"${key}" "d:\\Go\\src\\cftpweb\\candidateserver\\web\\*.tsx" "d:\\Go\\src\\cftpweb\\candidateserver\\web\\*.ts"`;
    try {
      execSync(cmd, { stdio: 'ignore' });
    } catch (e) {
      unused.push(key);
    }
  } catch(e) {}
}

console.log('Unused keys:', unused);

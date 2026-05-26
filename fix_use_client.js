const fs = require('fs');
const path = require('path');

function walk(dir) {
    let res = [];
    const list = fs.readdirSync(dir);
    list.forEach(f => {
        const file = path.join(dir, f);
        const stat = fs.statSync(file);
        if (stat.isDirectory() && f !== 'node_modules' && f !== '.next') {
            res = res.concat(walk(file));
        } else if (file.endsWith('.tsx') || file.endsWith('.ts')) {
            res.push(file);
        }
    });
    return res;
}

const files = [...walk('adminserver/web'), ...walk('candidateserver/web')];
let count = 0;
files.forEach(f => {
    let c = fs.readFileSync(f, 'utf8');
    
    // If it has use client but it's not the very first thing (ignoring whitespace)
    const trimmed = c.trimStart();
    if (trimmed.includes('"use client"') || trimmed.includes("'use client'")) {
        if (!trimmed.startsWith('"use client"') && !trimmed.startsWith("'use client'")) {
            // Remove all occurrences
            c = c.replace(/['"]use client['"]\r?\n?/g, '');
            // Prepend
            c = '"use client"\n' + c.trimStart();
            fs.writeFileSync(f, c);
            count++;
        }
    }
});
console.log('Fixed ' + count + ' files.');

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
        } else if (file.endsWith('.tsx')) {
            res.push(file);
        }
    });
    return res;
}

const files = [...walk('adminserver/web'), ...walk('candidateserver/web')];
files.forEach(f => {
    let c = fs.readFileSync(f, 'utf8');
    if (!c.includes('import React')) {
        if (c.includes('"use client"')) {
            c = c.replace(/"use client"(\r?\n)+/, '"use client"\nimport React from "react"\n');
        } else {
            c = 'import React from "react"\n' + c;
        }
        fs.writeFileSync(f, c);
    }
});
console.log('Done modifying ' + files.length + ' files.');

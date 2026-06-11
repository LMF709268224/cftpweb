const fs = require('fs');
let content = fs.readFileSync('d:/Go/src/cftpweb/candidateserver/web/app/membership/page.tsx', 'utf8');

content = content.replace(/features:\s+lang === "zh"\s*\?\s*\[.*?\]\s*:\s*\[.*?\],/gs, (match, offset) => {
  if (offset < 2000) return 'features: t.membership.basicBenefits,';
  if (offset < 2500) return 'features: t.membership.certifiedBenefits,';
  return 'features: t.membership.premiumBenefits,';
});

fs.writeFileSync('d:/Go/src/cftpweb/candidateserver/web/app/membership/page.tsx', content);

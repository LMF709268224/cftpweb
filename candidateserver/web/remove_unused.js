const fs = require('fs');

function cleanFile(filePath) {
  let content = fs.readFileSync(filePath, 'utf8');

  // remove keys from home
  content = content.replace(/\s+reapply:\s*".*?",/g, "");
  content = content.replace(/\s+todo1Title:\s*".*?",/g, "");
  content = content.replace(/\s+todo2Title:\s*".*?",/g, "");
  content = content.replace(/\s+todo2Desc:\s*".*?",/g, "");
  content = content.replace(/\s+todo3Title:\s*".*?",/g, "");
  content = content.replace(/\s+todo3Desc:\s*".*?",/g, "");

  // remove keys from courses
  content = content.replace(/\s+unit:\s*".*?",/g, "");
  content = content.replace(/\s+categoryCourse:\s*".*?",/g, "");
  content = content.replace(/\s+categoryColumn:\s*".*?",/g, "");
  content = content.replace(/\s+categoryShort:\s*".*?",/g, "");

  fs.writeFileSync(filePath, content);
}

cleanFile('d:/Go/src/cftpweb/candidateserver/web/lib/locales/zh.ts');
cleanFile('d:/Go/src/cftpweb/candidateserver/web/lib/locales/en.ts');

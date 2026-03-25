const fs = require("fs");
const path = require("path");

const rootDir = path.resolve(__dirname, "../..");
const rootReadme = path.join(rootDir, "README.md");
const backupReadme = path.join(rootDir, ".README.md.bak");
const npmReadme = path.join(rootDir, "npm", "README.md");

if (fs.existsSync(rootReadme)) {
  if (!fs.existsSync(backupReadme)) {
    fs.renameSync(rootReadme, backupReadme);
  }
}

if (fs.existsSync(npmReadme)) {
  fs.copyFileSync(npmReadme, rootReadme);
}

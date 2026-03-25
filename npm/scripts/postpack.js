const fs = require("fs");
const path = require("path");

const rootDir = path.resolve(__dirname, "../..");
const rootReadme = path.join(rootDir, "README.md");
const backupReadme = path.join(rootDir, ".README.md.bak");

if (fs.existsSync(backupReadme)) {
  if (fs.existsSync(rootReadme)) {
    fs.unlinkSync(rootReadme);
  }
  fs.renameSync(backupReadme, rootReadme);
}

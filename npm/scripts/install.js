'use strict';

const fs = require('fs');
const path = require('path');
const https = require('https');
const { spawnSync } = require('child_process');

let AdmZip = null;
try {
  AdmZip = require('adm-zip');
} catch (_) {
  AdmZip = null;
}

const LATEST_URL = process.env.CLI_TOP_NPM_LATEST_URL || 'https://cli-top.acmvit.in/latest.json';
const SKIP_DOWNLOAD = process.env.CLI_TOP_SKIP_DOWNLOAD === '1';

const ROOT_DIR = path.join(__dirname, '..');
const NATIVE_DIR = path.join(ROOT_DIR, 'bin', 'native');
const BIN_NAME = process.platform === 'win32' ? 'cli-top.exe' : 'cli-top';
const BIN_PATH = path.join(NATIVE_DIR, BIN_NAME);
const VERSION_PATH = path.join(NATIVE_DIR, '.version');

function getBinaryPath() {
  return BIN_PATH;
}

async function ensureBinary(options = {}) {
  const quiet = Boolean(options.quiet);
  if (SKIP_DOWNLOAD) {
    if (fs.existsSync(BIN_PATH)) {
      return BIN_PATH;
    }
    throw new Error('CLI_TOP_SKIP_DOWNLOAD=1 but binary is missing.');
  }

  const latest = await fetchJson(LATEST_URL);
  const desiredVersion = latest && latest.version ? String(latest.version) : '';

  if (desiredVersion && fs.existsSync(BIN_PATH) && fs.existsSync(VERSION_PATH)) {
    const installedVersion = fs.readFileSync(VERSION_PATH, 'utf8').trim();
    if (installedVersion === desiredVersion) {
      if (!quiet) {
        console.log(`cli-top ${installedVersion} already installed.`);
      }
      return BIN_PATH;
    }
  }

  const candidates = resolveDownloadCandidates(latest || {});
  if (candidates.length === 0) {
    throw new Error('No download URLs available in latest.json');
  }

  if (!quiet) {
    console.log(`Downloading cli-top ${desiredVersion || ''}...`.trim());
  }

  let lastErr;
  for (const url of candidates) {
    try {
      await downloadAndInstall(url, desiredVersion);
      if (!quiet) {
        console.log('cli-top installed successfully.');
      }
      return BIN_PATH;
    } catch (err) {
      lastErr = err;
      if (!quiet) {
        console.warn(`Failed to download from ${url}: ${err.message || err}`);
      }
    }
  }

  throw lastErr || new Error('Failed to download cli-top');
}

function resolveDownloadCandidates(latest) {
  const seen = new Set();
  const urls = [];
  const add = (url) => {
    if (!url || seen.has(url)) {
      return;
    }
    seen.add(url);
    urls.push(url);
  };

  const platform = process.platform;
  const arch = process.arch;
  const goos = platform === 'win32' ? 'windows' : platform;
  const baseUrl = normalizeBaseUrl(latest);

  if (latest.downloads) {
    add(resolveUrl(latest.downloads[`${goos}-${arch}`], baseUrl));
    add(resolveUrl(latest.downloads[`${platform}-${arch}`], baseUrl));
    add(resolveUrl(latest.downloads[goos], baseUrl));
  }

  switch (goos) {
    case 'windows':
      add(resolveUrl(latest.windowsUrl, baseUrl));
      break;
    case 'linux':
      add(resolveUrl(latest.linuxUrl, baseUrl));
      break;
    case 'darwin':
      add(resolveUrl(latest.macUrl, baseUrl));
      break;
    case 'android':
      add(resolveUrl(latest.androidUrl, baseUrl));
      break;
    default:
      break;
  }

  if (baseUrl && latest.version) {
    add(defaultBaseDownload(baseUrl, goos, String(latest.version)));
  }

  if (latest.version) {
    add(defaultLegacyDownload(goos, String(latest.version)));
  }

  return urls;
}

function normalizeBaseUrl(latest) {
  if (!latest || !latest.baseUrl) {
    return '';
  }
  try {
    const resolved = new URL(String(latest.baseUrl), LATEST_URL).toString();
    return resolved.replace(/\/$/, '');
  } catch (_) {
    return String(latest.baseUrl).replace(/\/$/, '');
  }
}

function resolveUrl(url, baseUrl) {
  if (!url) {
    return '';
  }
  const base = baseUrl || LATEST_URL;
  try {
    return new URL(String(url), base).toString();
  } catch (_) {
    return url;
  }
}

function defaultBaseDownload(baseUrl, goos, version) {
  if (!baseUrl || !version) {
    return '';
  }
  switch (goos) {
    case 'windows':
      return `${baseUrl}/v${version}/cli-top-windows-installer_v${version}.exe`;
    case 'linux':
      return `${baseUrl}/v${version}/cli-top-linux_v${version}.zip`;
    case 'android':
      return `${baseUrl}/v${version}/cli-top-android_v${version}.zip`;
    case 'darwin':
      return `${baseUrl}/v${version}/cli-top-macos_v${version}.zip`;
    default:
      return '';
  }
}

function defaultLegacyDownload(goos, version) {
  const base = 'https://raw.githubusercontent.com/technical-director-acmvit/cli-top-website/main/buildFiles';
  switch (goos) {
    case 'windows':
      return `${base}/v${version}/cli-top-windows-installer_v${version}.exe`;
    case 'linux':
      return `${base}/v${version}/cli-top-linux_v${version}.zip`;
    case 'android':
      return `${base}/v${version}/cli-top-android_v${version}.zip`;
    case 'darwin':
      return `${base}/v${version}/cli-top-macos_v${version}.zip`;
    default:
      return '';
  }
}

function downloadAndInstall(url, version) {
  return download(url).then((data) => {
    const isZip = looksLikeZip(data) || url.endsWith('.zip');
    fs.mkdirSync(NATIVE_DIR, { recursive: true });

    if (isZip) {
      if (AdmZip) {
        const zip = new AdmZip(data);
        const entry = pickBinaryEntry(zip.getEntries());
        if (!entry) {
          throw new Error('zip archive missing cli-top binary');
        }
        fs.writeFileSync(BIN_PATH, entry.getData());
      } else {
        const tmpZip = path.join(NATIVE_DIR, 'cli-top-download.zip');
        fs.writeFileSync(tmpZip, data);
        extractZip(tmpZip, NATIVE_DIR);
        const found = findBinaryFile(NATIVE_DIR);
        if (!found) {
          throw new Error('zip archive missing cli-top binary');
        }
        fs.copyFileSync(found, BIN_PATH);
        fs.unlinkSync(tmpZip);
      }
    } else {
      fs.writeFileSync(BIN_PATH, data);
    }

    if (process.platform !== 'win32') {
      fs.chmodSync(BIN_PATH, 0o755);
    }

    if (version) {
      fs.writeFileSync(VERSION_PATH, version, 'utf8');
    }
  });
}

function extractZip(zipPath, destDir) {
  if (process.platform === 'win32') {
    const command = `Expand-Archive -Path '${zipPath}' -DestinationPath '${destDir}' -Force`;
    const result = spawnSync('powershell', ['-NoProfile', '-Command', command], { stdio: 'inherit' });
    if (result.status !== 0) {
      throw new Error('failed to extract zip via PowerShell');
    }
    return;
  }

  const unzip = spawnSync('unzip', ['-o', zipPath, '-d', destDir], { stdio: 'inherit' });
  if (unzip.status === 0) {
    return;
  }

  const bsdtar = spawnSync('bsdtar', ['-xf', zipPath, '-C', destDir], { stdio: 'inherit' });
  if (bsdtar.status === 0) {
    return;
  }

  const tar = spawnSync('tar', ['-xf', zipPath, '-C', destDir], { stdio: 'inherit' });
  if (tar.status === 0) {
    return;
  }

  throw new Error('failed to extract zip (no unzip/bsdtar/tar found)');
}

function findBinaryFile(dir) {
  const entries = fs.readdirSync(dir, { withFileTypes: true });
  let fallback = null;
  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      const found = findBinaryFile(fullPath);
      if (found) {
        return found;
      }
      continue;
    }
    const name = entry.name.toLowerCase();
    if (name === 'cli-top' || name === 'cli-top.exe') {
      return fullPath;
    }
    if (!fallback && name.includes('cli-top')) {
      fallback = fullPath;
    }
  }
  return fallback;
}

function pickBinaryEntry(entries) {
  if (!entries || entries.length === 0) {
    return null;
  }
  const normalized = (entry) => entry.entryName.replace(/\\/g, '/');
  const exact = entries.find((entry) => /(^|\/)cli-top(\.exe)?$/i.test(normalized(entry)) && !entry.isDirectory);
  if (exact) {
    return exact;
  }
  const fallback = entries.find((entry) => !entry.isDirectory && /cli-top/i.test(normalized(entry)));
  if (fallback) {
    return fallback;
  }
  return entries.find((entry) => !entry.isDirectory) || null;
}

function looksLikeZip(buffer) {
  return buffer && buffer.length >= 4 && buffer[0] === 0x50 && buffer[1] === 0x4b;
}

function fetchJson(url) {
  return download(url).then((data) => {
    try {
      return JSON.parse(data.toString('utf8'));
    } catch (err) {
      throw new Error(`Failed to parse JSON from ${url}`);
    }
  });
}

function download(url, redirects = 0) {
  return new Promise((resolve, reject) => {
    const req = https.get(url, (res) => {
      const status = res.statusCode || 0;
      if (status >= 300 && status < 400 && res.headers.location) {
        if (redirects > 5) {
          res.resume();
          reject(new Error('Too many redirects'));
          return;
        }
        res.resume();
        resolve(download(res.headers.location, redirects + 1));
        return;
      }
      if (status >= 400) {
        res.resume();
        reject(new Error(`Request failed with status ${status}`));
        return;
      }
      const chunks = [];
      res.on('data', (chunk) => chunks.push(chunk));
      res.on('end', () => resolve(Buffer.concat(chunks)));
    });
    req.on('error', reject);
  });
}

module.exports = { ensureBinary, getBinaryPath };

if (require.main === module) {
  ensureBinary({ quiet: false }).catch((err) => {
    console.error(err && err.message ? err.message : err);
    process.exit(1);
  });
}

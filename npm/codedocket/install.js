#!/usr/bin/env node
"use strict";

const childProcess = require("node:child_process");
const fs = require("node:fs");
const https = require("node:https");
const os = require("node:os");
const path = require("node:path");

const pkg = require("./package.json");

const repo = "Amaan-Khan14/codedocket";
const version = process.env.CODEDOCKET_INSTALL_VERSION || `v${pkg.version}`;
const platform = mapPlatform(process.platform);
const arch = mapArch(process.arch);
const archiveExt = platform === "windows" ? "zip" : "tar.gz";
const archiveName = `codedocket_${platform}_${arch}.${archiveExt}`;
const archiveUrl = `https://github.com/${repo}/releases/download/${version}/${archiveName}`;
const vendorDir = path.join(__dirname, "vendor");
const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "codedocket-npm-"));
const archivePath = path.join(tmpDir, archiveName);
const extractDir = path.join(tmpDir, "extract");
const exe = platform === "windows" ? "codedocket.exe" : "codedocket";

main().catch((err) => {
  console.error(`CodeDocket install failed: ${err.message}`);
  process.exit(1);
});

async function main() {
  fs.mkdirSync(vendorDir, { recursive: true });
  fs.mkdirSync(extractDir, { recursive: true });

  await download(archiveUrl, archivePath);
  extractArchive(archivePath, extractDir, archiveExt);

  const binary = findFile(extractDir, exe);
  if (!binary) {
    throw new Error(`downloaded archive did not contain ${exe}`);
  }

  const target = path.join(vendorDir, exe);
  fs.copyFileSync(binary, target);
  if (platform !== "windows") {
    fs.chmodSync(target, 0o755);
  }

  fs.rmSync(tmpDir, { recursive: true, force: true });
}

function mapPlatform(value) {
  switch (value) {
    case "darwin":
      return "darwin";
    case "linux":
      return "linux";
    case "win32":
      return "windows";
    default:
      throw new Error(`unsupported platform: ${value}`);
  }
}

function mapArch(value) {
  switch (value) {
    case "x64":
      return "amd64";
    case "arm64":
      return "arm64";
    default:
      throw new Error(`unsupported architecture: ${value}`);
  }
}

function download(url, destination) {
  return new Promise((resolve, reject) => {
    const request = https.get(url, (response) => {
      if ([301, 302, 303, 307, 308].includes(response.statusCode)) {
        response.resume();
        download(response.headers.location, destination).then(resolve, reject);
        return;
      }

      if (response.statusCode !== 200) {
        response.resume();
        reject(new Error(`download ${url} returned HTTP ${response.statusCode}`));
        return;
      }

      const file = fs.createWriteStream(destination);
      response.pipe(file);
      file.on("finish", () => file.close(resolve));
      file.on("error", reject);
    });

    request.on("error", reject);
  });
}

function extractArchive(archive, destination, ext) {
  if (ext === "tar.gz") {
    run("tar", ["-xzf", archive, "-C", destination]);
    return;
  }

  const script = `Expand-Archive -LiteralPath ${quotePowerShell(archive)} -DestinationPath ${quotePowerShell(destination)} -Force`;
  run("powershell.exe", ["-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script]);
}

function run(command, args) {
  const result = childProcess.spawnSync(command, args, { stdio: "inherit" });
  if (result.error) {
    throw result.error;
  }
  if (result.status !== 0) {
    throw new Error(`${command} exited with status ${result.status}`);
  }
}

function findFile(dir, name) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const fullPath = path.join(dir, entry.name);
    if (entry.isFile() && entry.name === name) {
      return fullPath;
    }
    if (entry.isDirectory()) {
      const found = findFile(fullPath, name);
      if (found) {
        return found;
      }
    }
  }
  return "";
}

function quotePowerShell(value) {
  return `'${value.replace(/'/g, "''")}'`;
}

#!/usr/bin/env node
"use strict";

const childProcess = require("node:child_process");
const fs = require("node:fs");
const path = require("node:path");

const exe = process.platform === "win32" ? "codedocket.exe" : "codedocket";
const binary = path.join(__dirname, "..", "vendor", exe);

if (!fs.existsSync(binary)) {
  console.error("CodeDocket binary is missing. Reinstall the package to run postinstall again.");
  process.exit(1);
}

const result = childProcess.spawnSync(binary, process.argv.slice(2), {
  stdio: "inherit",
  windowsHide: false
});

if (result.error) {
  console.error(result.error.message);
  process.exit(1);
}

process.exit(result.status === null ? 1 : result.status);

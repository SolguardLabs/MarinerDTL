import { readFileSync } from "node:fs";
import { extname } from "node:path";
import { spawnSync } from "node:child_process";

const listed = spawnSync("git", ["ls-files", "-co", "--exclude-standard"], {
  encoding: "utf8",
  shell: false,
});
if (listed.status !== 0) {
  process.stderr.write(listed.stderr);
  process.exit(listed.status ?? 1);
}

const textExtensions = new Set([".go", ".md", ".mjs", ".ts", ".json", ".yml", ".yaml", ".sh"]);
const forbidden = [
  /\bctf\b/i,
  /\blaborator(?:io|y|ies)\w*\b/i,
  /\bvulnerab\w*\b/i,
  /\bexploit\w*\b/i,
];
const findings = [];

for (const file of listed.stdout.split(/\r?\n/).filter(Boolean)) {
  if (
    file === "LICENSE" ||
    file === "scripts/scan-public.mjs" ||
    !textExtensions.has(extname(file))
  ) {
    continue;
  }
  const lines = readFileSync(file, "utf8").split(/\r?\n/);
  lines.forEach((line, index) => {
    if (forbidden.some((pattern) => pattern.test(line))) {
      findings.push(`${file}:${index + 1}`);
    }
  });
}

if (findings.length > 0) {
  console.error(`Public terminology check failed:\n${findings.join("\n")}`);
  process.exit(1);
}
console.log("public terminology check: ok");

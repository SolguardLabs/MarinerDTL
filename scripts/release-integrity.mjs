import { createHash } from "node:crypto";
import { existsSync, readFileSync, statSync } from "node:fs";
import { spawnSync } from "node:child_process";

const required = [
  "README.md",
  "SECURITY.md",
  "CHANGELOG.md",
  "CONTRIBUTING.md",
  "assets/banner.png",
  "docs/arquitectura.md",
  "docs/modelo-economico.md",
  "docs/riesgo-y-capital.md",
  "docs/operacion.md",
  "docs/integracion.md",
];

for (const file of required) {
  if (!existsSync(file) || statSync(file).size === 0) {
    console.error(`Missing release artifact: ${file}`);
    process.exit(1);
  }
}

const manifest = {
  "src/engine/service.go": "23e08e613552958e8fb97e6118f58f1e54d565e1",
  "src/settlement/rebate.go": "c71c7000dd99505b66522aabd87acfbce3a21414",
};
for (const [file, expected] of Object.entries(manifest)) {
  const result = spawnSync("git", ["hash-object", file], { encoding: "utf8", shell: false });
  if (result.status !== 0 || result.stdout.trim() !== expected) {
    console.error(`Protected source mismatch: ${file}`);
    process.exit(1);
  }
}

const pkg = JSON.parse(readFileSync("package.json", "utf8"));
if (pkg.version !== "1.0.0") {
  console.error(`Unexpected package version: ${pkg.version}`);
  process.exit(1);
}
if (!readFileSync("README.md", "utf8").includes("assets/banner.png")) {
  console.error("README does not reference the canonical banner");
  process.exit(1);
}

const banner = readFileSync("assets/banner.png");
if (banner.length < 100_000 || banner.subarray(1, 4).toString("ascii") !== "PNG") {
  console.error("Banner is not a valid production PNG");
  process.exit(1);
}
const digest = createHash("sha256").update(banner).digest("hex");

const ref = process.env.GITHUB_REF_NAME ?? "";
if (process.env.GITHUB_REF_TYPE === "tag" && ref !== `v${pkg.version}`) {
  console.error(`Tag ${ref} does not match package version ${pkg.version}`);
  process.exit(1);
}
console.log(`release integrity: ok (${digest})`);

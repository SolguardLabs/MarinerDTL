import { spawnSync } from "node:child_process";

const command = process.platform === "win32" ? "gofmt.exe" : "gofmt";
const result = spawnSync(command, ["-l", "src"], { encoding: "utf8", shell: false });

if (result.error) {
  console.error(result.error.message);
  process.exit(1);
}
if (result.status !== 0) {
  process.stderr.write(result.stderr);
  process.exit(result.status ?? 1);
}
if (result.stdout.trim()) {
  console.error("Go files require gofmt:");
  console.error(result.stdout.trim());
  process.exit(1);
}

console.log("gofmt check: ok");

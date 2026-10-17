import { spawnSync } from "node:child_process";

const go = process.platform === "win32" ? "go.exe" : "go";
const npmCommand = process.env.npm_execpath ? process.execPath : "npm";
const npmPrefix = process.env.npm_execpath ? [process.env.npm_execpath] : [];
const npmArgs = (...args) => [...npmPrefix, ...args];
const commands = [
  [process.execPath, ["scripts/gofmt-check.mjs"]],
  [go, ["test", "./..."]],
  [go, ["vet", "./..."]],
  [npmCommand, npmArgs("run", "build")],
  [npmCommand, npmArgs("run", "check")],
  [npmCommand, npmArgs("test")],
  [npmCommand, npmArgs("run", "format")],
  [npmCommand, npmArgs("run", "loc")],
  [npmCommand, npmArgs("run", "scan:public")],
  [npmCommand, npmArgs("run", "release:check")],
];

for (const [command, args] of commands) {
  console.log(`\n> ${command} ${args.join(" ")}`);
  const result = spawnSync(command, args, { stdio: "inherit", shell: false });
  if (result.error) {
    console.error(result.error.message);
    process.exit(1);
  }
  if (result.status !== 0) {
    process.exit(result.status ?? 1);
  }
}

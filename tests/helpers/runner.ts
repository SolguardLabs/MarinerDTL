import { spawnSync } from "node:child_process";
import { existsSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

export type AmountEntry = {
  id: string;
  amount: number;
};

export type AccountReport = {
  id: string;
  role: string;
  displayName?: string;
  free: AmountEntry[];
  reserved: AmountEntry[];
  settled: AmountEntry[];
  rebates: AmountEntry[];
  penalties: AmountEntry[];
};

export type RouteReport = {
  id: string;
  asset: string;
  shipperId: string;
  operatorId: string;
  rebateAccountId: string;
  status: string;
  escrowTotal: number;
  escrowRemaining: number;
  rebateBudget: number;
  rebateClaimed: number;
  deliveredBps: number;
  releasedAmount: number;
  cancelledAmount: number;
  penaltyAmount: number;
  frozenAmount: number;
  milestones: string[];
};

export type MilestoneReport = {
  id: string;
  routeId: string;
  sequence: number;
  leg: string;
  carrierId: string;
  custodianId: string;
  amount: number;
  status: string;
  certificateId?: string;
  disputeId?: string;
  frozenAmount: number;
  releasedAmount: number;
  cancelledAmount: number;
  penaltyAmount: number;
  remainingAmount: number;
};

export type DisputeReport = {
  id: string;
  routeId: string;
  milestoneId: string;
  claimantId: string;
  status: string;
  amountFrozen: number;
  penaltyBps: number;
  penaltyCharged: number;
};

export type MarinerReport = {
  name: string;
  epoch: number;
  accounts: AccountReport[];
  routes: RouteReport[];
  milestones: MilestoneReport[];
  disputes: DisputeReport[];
  metrics: AmountEntry[];
  auditIssues: Array<{ code: string; severity: string; message: string }>;
};

export const root = resolve(dirname(fileURLToPath(import.meta.url)), "..", "..");
export const binary = join(
  root,
  "bin",
  process.platform === "win32" ? "marinerdtl.exe" : "marinerdtl",
);

export function ensureBuilt(): void {
  if (existsSync(binary)) {
    return;
  }
  const result = spawnSync(process.execPath, ["scripts/build.mjs"], {
    cwd: root,
    encoding: "utf8",
  });
  if (result.status !== 0) {
    throw new Error(result.stderr || result.stdout || "build failed");
  }
}

export function runCli(args: string[]): string {
  ensureBuilt();
  const result = spawnSync(binary, args, {
    cwd: root,
    encoding: "utf8",
  });
  if (result.status !== 0) {
    throw new Error(`command failed: ${binary} ${args.join(" ")}\n${result.stderr}`);
  }
  return result.stdout;
}

export function runFixture(name: string): MarinerReport {
  return JSON.parse(runCli(["run", join("tests", "fixtures", name), "--json"])) as MarinerReport;
}

export function validateFixture(name: string): { status: string; issues: unknown[] } {
  return JSON.parse(runCli(["validate", join("tests", "fixtures", name), "--json"])) as {
    status: string;
    issues: unknown[];
  };
}

export function byId<T extends { id: string }>(items: T[], id: string): T {
  const found = items.find((item) => item.id === id);
  if (!found) {
    throw new Error(`missing ${id}`);
  }
  return found;
}

export function amount(entries: AmountEntry[], id: string): number {
  return entries.find((entry) => entry.id === id)?.amount ?? 0;
}

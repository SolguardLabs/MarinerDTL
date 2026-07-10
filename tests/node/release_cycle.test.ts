import assert from "node:assert/strict";
import test from "node:test";
import { amount, byId, runFixture } from "../helpers/runner.ts";

test("certified route releases all milestones and pays route rebate", () => {
  const report = runFixture("release_cycle.json");
  const route = byId(report.routes, "pacific-777");
  const carrier = byId(report.accounts, "carrier-azul");
  const operator = byId(report.accounts, "operator-quay");
  const treasury = byId(report.accounts, "treasury-main");

  assert.equal(route.status, "completed");
  assert.equal(route.deliveredBps, 10000);
  assert.equal(route.releasedAmount, 18000);
  assert.equal(route.escrowRemaining, 0);
  assert.equal(route.rebateClaimed, 900);
  assert.equal(amount(carrier.settled, "usdc"), 18000);
  assert.equal(amount(operator.rebates, "usdc"), 900);
  assert.equal(amount(treasury.free, "usdc"), 49100);
  assert.deepEqual(report.auditIssues, []);
});

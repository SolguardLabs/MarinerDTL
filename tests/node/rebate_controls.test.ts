import assert from "node:assert/strict";
import test from "node:test";
import { amount, byId, runFixture } from "../helpers/runner.ts";

test("rebate remains unavailable until route progress reaches threshold", () => {
  const report = runFixture("rebate_threshold.json");
  const route = byId(report.routes, "med-119");
  const operator = byId(report.accounts, "operator-quay");

  assert.equal(route.status, "in_transit");
  assert.equal(route.deliveredBps, 2500);
  assert.equal(route.rebateClaimed, 0);
  assert.equal(amount(operator.rebates, "usdc"), 0);
  assert.deepEqual(report.auditIssues, []);
});

import assert from "node:assert/strict";
import test from "node:test";
import { amount, byId, runFixture, validateFixture } from "../helpers/runner.ts";

test("route escrow locks shipper balance and staged milestones", () => {
  const report = runFixture("escrow_funding.json");
  const route = byId(report.routes, "atlantic-001");
  const shipper = byId(report.accounts, "shipper-norte");

  assert.equal(route.status, "in_transit");
  assert.equal(route.escrowTotal, 12000);
  assert.equal(route.escrowRemaining, 12000);
  assert.equal(route.milestones.length, 3);
  assert.equal(amount(shipper.free, "usdc"), 3000);
  assert.equal(amount(shipper.reserved, "usdc"), 12000);
  assert.deepEqual(report.auditIssues, []);
});

test("fixture validation returns structured status", () => {
  const result = validateFixture("escrow_funding.json");
  assert.equal(result.status, "ok");
  assert.deepEqual(result.issues, []);
});

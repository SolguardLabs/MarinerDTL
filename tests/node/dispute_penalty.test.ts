import assert from "node:assert/strict";
import test from "node:test";
import { amount, byId, runFixture } from "../helpers/runner.ts";

test("partial dispute freezes the affected milestone until cancellation", () => {
  const report = runFixture("dispute_penalty.json");
  const route = byId(report.routes, "baltic-204");
  const main = byId(report.milestones, "baltic-main-leg");
  const feeder = byId(report.milestones, "baltic-feeder-leg");
  const dispute = byId(report.disputes, "disp-baltic-main");
  const shipper = byId(report.accounts, "shipper-norte");
  const carrier = byId(report.accounts, "carrier-azul");

  assert.equal(feeder.status, "released");
  assert.equal(main.status, "cancelled");
  assert.equal(main.cancelledAmount, 8550);
  assert.equal(main.penaltyAmount, 450);
  assert.equal(dispute.status, "open");
  assert.equal(dispute.amountFrozen, 9000);
  assert.equal(route.status, "cancelled");
  assert.equal(route.releasedAmount, 3000);
  assert.equal(route.cancelledAmount, 8550);
  assert.equal(route.penaltyAmount, 450);
  assert.equal(amount(shipper.free, "usdc"), 11550);
  assert.equal(amount(carrier.penalties, "usdc"), 450);
  assert.deepEqual(report.auditIssues, []);
});

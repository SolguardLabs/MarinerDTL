import assert from "node:assert/strict";
import test from "node:test";
import { analyzeFixture } from "../helpers/runner.ts";

test("risk report reconciles gross and at-risk route values", () => {
  const report = analyzeFixture("release_cycle.json", 100_000);

  assert.equal(report.scenario, "operating");
  assert.equal(report.grossValue, 18_000);
  assert.equal(report.routes.length, 1);
  assert.equal(report.routes[0]?.routeId, "pacific-777");
  assert.equal(report.atRiskValue, report.routes[0]?.atRiskValue);
  assert.match(report.digest, /^[a-f0-9]{64}$/);
});

test("severe profile increases the route capital charge", () => {
  const operating = analyzeFixture("escrow_funding.json", 100_000);
  const severe = analyzeFixture("escrow_funding.json", 100_000, true);

  assert.equal(severe.scenario, "severe");
  assert.ok(severe.routes[0]!.capitalCharge > operating.routes[0]!.capitalCharge);
  assert.ok(severe.capital.requiredCapital > operating.capital.requiredCapital);
});

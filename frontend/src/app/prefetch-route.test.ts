import { describe, expect, it } from "vitest";
import { getRoutePrefetchKeys } from "./prefetch-route";

describe("getRoutePrefetchKeys", () => {
  it("maps public, user, and admin paths to prefetchable route chunks", () => {
    expect(getRoutePrefetchKeys("/")).toEqual(["landing"]);
    expect(getRoutePrefetchKeys("/#features")).toEqual(["landing"]);
    expect(getRoutePrefetchKeys("/stats")).toEqual(["stats"]);
    expect(getRoutePrefetchKeys("/dashboard/domains")).toEqual(["userDomains"]);
    expect(getRoutePrefetchKeys("/dashboard/settings")).toEqual(["userSettings"]);
    expect(getRoutePrefetchKeys("/admin")).toEqual(["adminOverview"]);
    expect(getRoutePrefetchKeys("/admin/jobs")).toEqual(["adminJobs"]);
    expect(getRoutePrefetchKeys("/admin/account")).toEqual(["adminAccount"]);
    expect(getRoutePrefetchKeys("/unknown")).toEqual([]);
  });
});

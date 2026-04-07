import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { AdminAccountPage } from "./account-page";

vi.mock("@/features/account/pages/account-settings-page", () => ({
  AccountSettingsPage: ({ consoleKind }: { consoleKind: "user" | "admin" }) => (
    <div>account-settings:{consoleKind}</div>
  ),
}));

describe("AdminAccountPage", () => {
  it("renders the shared account settings page in admin mode", () => {
    render(<AdminAccountPage />);
    expect(screen.getByText("account-settings:admin")).toBeInTheDocument();
  });
});

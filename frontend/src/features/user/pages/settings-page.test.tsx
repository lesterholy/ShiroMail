import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { UserSettingsPage } from "./settings-page";

vi.mock("@/features/account/pages/account-settings-page", () => ({
  AccountSettingsPage: ({ consoleKind }: { consoleKind: "user" | "admin" }) => (
    <div>account-settings:{consoleKind}</div>
  ),
}));

describe("UserSettingsPage", () => {
  it("renders the shared account settings page in user mode", () => {
    render(<UserSettingsPage />);
    expect(screen.getByText("account-settings:user")).toBeInTheDocument();
  });
});

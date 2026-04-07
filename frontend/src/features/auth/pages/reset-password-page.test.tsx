import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { resendEmailVerification, resetPassword } from "../api";
import { ResetPasswordPage } from "./reset-password-page";

vi.mock("../api", () => ({
  resetPassword: vi.fn(),
  resendEmailVerification: vi.fn(),
  getAuthErrorMessage: (error: unknown, fallback: string) =>
    error instanceof Error ? error.message : fallback,
}));

describe("ResetPasswordPage", () => {
  beforeEach(() => {
    cleanup();
    vi.clearAllMocks();
    vi.mocked(resetPassword).mockResolvedValue({ status: "ok" });
    vi.mocked(resendEmailVerification).mockResolvedValue({
      status: "verification_required",
      email: "reset-user@example.com",
      verificationTicket: "ticket-2",
      expiresInSeconds: 900,
    });
  });

  function renderPage(initialEntry = "/auth/reset-password?ticket=ticket-1&email=reset-user@example.com&code=654321") {
    render(
      <MemoryRouter initialEntries={[initialEntry]}>
        <Routes>
          <Route path="/auth/reset-password" element={<ResetPasswordPage />} />
          <Route path="/" element={<div>home</div>} />
        </Routes>
      </MemoryRouter>,
    );
  }

  it("prefills code from the reset password link and submits reset", async () => {
    renderPage();

    expect(screen.getByLabelText("重置验证码")).toHaveValue("654321");
    fireEvent.change(screen.getByLabelText("新密码"), {
      target: { value: "BetterSecret456!" },
    });
    fireEvent.click(screen.getByRole("button", { name: "确认重置" }));

    await waitFor(() => {
      expect(vi.mocked(resetPassword)).toHaveBeenCalledWith({
        verificationTicket: "ticket-1",
        code: "654321",
        newPassword: "BetterSecret456!",
      });
    });
  });

  it("resends the code and updates the active verification ticket", async () => {
    renderPage("/auth/reset-password?ticket=ticket-1&email=reset-user@example.com");

    fireEvent.click(screen.getByRole("button", { name: "重新发送验证码" }));

    await waitFor(() => {
      expect(vi.mocked(resendEmailVerification)).toHaveBeenCalledWith({
        verificationTicket: "ticket-1",
      });
    });

    fireEvent.change(screen.getByLabelText("重置验证码"), {
      target: { value: "111222" },
    });
    fireEvent.change(screen.getByLabelText("新密码"), {
      target: { value: "BetterSecret456!" },
    });
    fireEvent.click(screen.getByRole("button", { name: "确认重置" }));

    await waitFor(() => {
      expect(vi.mocked(resetPassword)).toHaveBeenCalledWith({
        verificationTicket: "ticket-2",
        code: "111222",
        newPassword: "BetterSecret456!",
      });
    });
  });
});

import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useAuthStore } from "@/lib/auth-store";
import { confirmEmailVerification, resendEmailVerification } from "../api";
import { VerifyEmailPage } from "./verify-email-page";

vi.mock("../api", () => ({
  confirmEmailVerification: vi.fn(),
  resendEmailVerification: vi.fn(),
  getAuthErrorMessage: (error: unknown, fallback: string) =>
    error instanceof Error ? error.message : fallback,
}));

describe("VerifyEmailPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useAuthStore.setState({
      accessToken: null,
      refreshToken: null,
      user: null,
    });
    vi.mocked(confirmEmailVerification).mockResolvedValue({
      accessToken: "token",
      refreshToken: "refresh",
      user: {
        userId: 1,
        username: "verify-user",
        roles: ["user"],
      },
    });
    vi.mocked(resendEmailVerification).mockResolvedValue({
      status: "verification_required",
      email: "verify-user@example.com",
      verificationTicket: "ticket-2",
      expiresInSeconds: 900,
    });
  });

  function renderPage(initialEntry = "/auth/verify-email?ticket=ticket-1&email=verify-user@example.com&code=123456") {
    render(
      <MemoryRouter initialEntries={[initialEntry]}>
        <Routes>
          <Route path="/auth/verify-email" element={<VerifyEmailPage />} />
          <Route path="/dashboard" element={<div>dashboard</div>} />
        </Routes>
      </MemoryRouter>,
    );
  }

  it("prefills the verification code from the email link", async () => {
    renderPage();

    expect(screen.getByLabelText("邮箱验证码")).toHaveValue("123456");
    fireEvent.click(screen.getByRole("button", { name: "确认验证" }));

    await waitFor(() => {
      expect(vi.mocked(confirmEmailVerification)).toHaveBeenCalledWith({
        verificationTicket: "ticket-1",
        code: "123456",
      });
    });
  });

  it("uses the latest ticket after resending verification code", async () => {
    renderPage("/auth/verify-email?ticket=ticket-1&email=verify-user@example.com");

    fireEvent.click(screen.getByRole("button", { name: "重新发送验证码" }));

    await waitFor(() => {
      expect(vi.mocked(resendEmailVerification)).toHaveBeenCalledWith({
        verificationTicket: "ticket-1",
      });
    });

    fireEvent.change(screen.getByLabelText("邮箱验证码"), {
      target: { value: "654321" },
    });
    fireEvent.click(screen.getByRole("button", { name: "确认验证" }));

    await waitFor(() => {
      expect(vi.mocked(confirmEmailVerification)).toHaveBeenCalledWith({
        verificationTicket: "ticket-2",
        code: "654321",
      });
    });
  });
});

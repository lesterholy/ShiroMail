import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useAuthStore } from "@/lib/auth-store";
import {
  changeAccountPassword,
  confirmAccountEmailChange,
  disableTOTP,
  enableTOTP,
  fetchAccountProfile,
  fetchTOTPStatus,
  requestAccountEmailChange,
  setupTOTP,
  updateAccountProfile,
} from "../api";
import { AccountSettingsPage } from "./account-settings-page";

vi.mock("../api", () => ({
  fetchAccountProfile: vi.fn(),
  updateAccountProfile: vi.fn(),
  requestAccountEmailChange: vi.fn(),
  confirmAccountEmailChange: vi.fn(),
  changeAccountPassword: vi.fn(),
  fetchTOTPStatus: vi.fn(),
  setupTOTP: vi.fn(),
  enableTOTP: vi.fn(),
  disableTOTP: vi.fn(),
  getAccountErrorMessage: (error: unknown, fallback: string) =>
    error instanceof Error ? error.message : fallback,
}));

describe("AccountSettingsPage", () => {
  beforeEach(() => {
    cleanup();
    vi.clearAllMocks();
    useAuthStore.setState({
      accessToken: "token-admin",
      refreshToken: "refresh-admin",
      user: {
        userId: 7,
        username: "galiais",
        roles: ["admin", "user"],
      },
    });
    Object.assign(navigator, {
      clipboard: {
        writeText: vi.fn().mockResolvedValue(undefined),
      },
    });

    vi.mocked(fetchAccountProfile).mockResolvedValue({
      userId: 7,
      username: "galiais",
      email: "galiais@example.com",
      emailVerified: true,
      roles: ["admin", "user"],
      displayName: "Galiais",
      locale: "zh-CN",
      timezone: "Asia/Shanghai",
      autoRefreshSeconds: 30,
      twoFactorEnabled: false,
    });
    vi.mocked(fetchTOTPStatus).mockResolvedValue({ enabled: false });
    vi.mocked(updateAccountProfile).mockResolvedValue({
      userId: 7,
      username: "galiais",
      email: "galiais@example.com",
      emailVerified: true,
      roles: ["admin", "user"],
      displayName: "Ops Galiais",
      locale: "en-US",
      timezone: "UTC",
      autoRefreshSeconds: 45,
      twoFactorEnabled: false,
    });
    vi.mocked(requestAccountEmailChange).mockResolvedValue({
      status: "verification_required",
      email: "ops@example.com",
      verificationTicket: "ticket-1",
      expiresInSeconds: 900,
    });
    vi.mocked(confirmAccountEmailChange).mockResolvedValue({
      userId: 7,
      username: "galiais",
      email: "ops@example.com",
      emailVerified: true,
      roles: ["admin", "user"],
      displayName: "Galiais",
      locale: "zh-CN",
      timezone: "Asia/Shanghai",
      autoRefreshSeconds: 30,
      twoFactorEnabled: false,
    });
    vi.mocked(changeAccountPassword).mockResolvedValue({ status: "ok" });
    vi.mocked(setupTOTP).mockResolvedValue({
      manualEntryKey: "SECRET123",
      otpauthUrl: "otpauth://totp/Shiro%20Email:galiais@example.com?secret=SECRET123",
    });
    vi.mocked(enableTOTP).mockResolvedValue({ status: "ok" });
    vi.mocked(disableTOTP).mockResolvedValue({ status: "ok" });
  });

  function renderPage() {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });

    render(
      <MemoryRouter>
        <QueryClientProvider client={queryClient}>
          <AccountSettingsPage consoleKind="admin" />
        </QueryClientProvider>
      </MemoryRouter>,
    );
  }

  it("renders account summary and profile fields", async () => {
    renderPage();

    expect(await screen.findByDisplayValue("galiais")).toBeInTheDocument();
    expect(screen.getByDisplayValue("galiais")).toBeInTheDocument();
    expect(screen.getAllByDisplayValue("galiais@example.com").length).toBeGreaterThan(0);
    expect(screen.getByText(/TOTP (inactive|未启用)/)).toBeInTheDocument();
  });

  it("updates account profile and requests email verification", async () => {
    renderPage();

    await screen.findByDisplayValue("galiais");

    fireEvent.change(screen.getByDisplayValue("Galiais"), {
      target: { value: "Ops Galiais" },
    });
    fireEvent.change(screen.getByDisplayValue("30"), {
      target: { value: "45" },
    });
    fireEvent.click(screen.getAllByRole("button", { name: /Save settings|保存设置/ })[0]);

    await waitFor(() => {
      expect(vi.mocked(updateAccountProfile).mock.calls[0]?.[0]).toEqual({
        displayName: "Ops Galiais",
        locale: "zh-CN",
        timezone: "Asia/Shanghai",
        autoRefreshSeconds: 45,
      });
    });

    fireEvent.change(screen.getByPlaceholderText("name@example.com"), {
      target: { value: "ops@example.com" },
    });
    fireEvent.click(screen.getByRole("button", { name: /Send code|发送验证码/ }));

    await waitFor(() => {
      expect(vi.mocked(requestAccountEmailChange).mock.calls[0]?.[0]).toBe("ops@example.com");
    });
    expect(await screen.findByText(/ops@example\.com/)).toBeInTheDocument();

    fireEvent.change(screen.getByPlaceholderText(/Enter the 6-digit code|输入 6 位验证码/), {
      target: { value: "654321" },
    });
    fireEvent.click(screen.getByRole("button", { name: /Confirm email change|确认换绑/ }));

    await waitFor(() => {
      expect(vi.mocked(confirmAccountEmailChange).mock.calls[0]?.[0]).toEqual({
        verificationTicket: "ticket-1",
        code: "654321",
      });
    });
  });

  it("prepares and enables totp", async () => {
    renderPage();

    await screen.findByDisplayValue("galiais");
    fireEvent.click(screen.getByRole("button", { name: /Prepare 2FA|开始配置 2FA/ }));

    expect(await screen.findByDisplayValue("SECRET123")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Copy manual key|复制手动密钥/ }));
    await waitFor(() => {
      expect(navigator.clipboard.writeText).toHaveBeenCalledWith("SECRET123");
    });
    fireEvent.change(screen.getByPlaceholderText(/Enter the 6-digit code|输入 6 位验证码/), {
      target: { value: "123456" },
    });
    fireEvent.click(screen.getByRole("button", { name: /Enable TOTP|启用 TOTP/ }));

    await waitFor(() => {
      expect(vi.mocked(enableTOTP).mock.calls[0]?.[0]).toBe("123456");
    });
  });

  it("disables totp with the current password when already enabled", async () => {
    vi.mocked(fetchTOTPStatus).mockResolvedValue({ enabled: true });
    vi.mocked(fetchAccountProfile).mockResolvedValue({
      userId: 7,
      username: "galiais",
      email: "galiais@example.com",
      emailVerified: true,
      roles: ["admin", "user"],
      displayName: "Galiais",
      locale: "zh-CN",
      timezone: "Asia/Shanghai",
      autoRefreshSeconds: 30,
      twoFactorEnabled: true,
    });

    renderPage();

    await screen.findByDisplayValue("galiais");
    fireEvent.change(screen.getAllByLabelText(/Current password|当前密码/)[1], {
      target: { value: "Secret123!" },
    });
    fireEvent.click(screen.getByRole("button", { name: /Disable TOTP|停用 TOTP/ }));

    await waitFor(() => {
      expect(vi.mocked(disableTOTP).mock.calls[0]?.[0]).toBe("Secret123!");
    });
  });

  it("shows a retry action when account profile loading fails", async () => {
    vi.mocked(fetchAccountProfile).mockRejectedValueOnce(new Error("load failed"));

    renderPage();

    expect(await screen.findByText(/Account settings|账户设置/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Refresh|刷新数据/ }));

    await waitFor(() => {
      expect(vi.mocked(fetchAccountProfile)).toHaveBeenCalledTimes(2);
    });
  });

  it("refetches account data when the authenticated user changes", async () => {
    vi.mocked(fetchAccountProfile)
      .mockResolvedValueOnce({
        userId: 7,
        username: "galiais",
        email: "galiais@example.com",
        emailVerified: true,
        roles: ["admin", "user"],
        displayName: "Galiais",
        locale: "zh-CN",
        timezone: "Asia/Shanghai",
        autoRefreshSeconds: 30,
        twoFactorEnabled: true,
      })
      .mockResolvedValueOnce({
        userId: 1,
        username: "alice",
        email: "alice@example.com",
        emailVerified: true,
        roles: ["user"],
        displayName: "Alice",
        locale: "zh-CN",
        timezone: "Asia/Shanghai",
        autoRefreshSeconds: 30,
        twoFactorEnabled: false,
      });
    vi.mocked(fetchTOTPStatus)
      .mockResolvedValueOnce({ enabled: true })
      .mockResolvedValueOnce({ enabled: false });

    renderPage();

    expect(await screen.findByDisplayValue("galiais")).toBeInTheDocument();

    useAuthStore.setState({
      accessToken: "token-alice",
      refreshToken: "refresh-alice",
      user: {
        userId: 1,
        username: "alice",
        roles: ["user"],
      },
    });

    expect(await screen.findByDisplayValue("alice")).toBeInTheDocument();
    await waitFor(() => {
      expect(vi.mocked(fetchAccountProfile)).toHaveBeenCalledTimes(2);
      expect(vi.mocked(fetchTOTPStatus)).toHaveBeenCalledTimes(2);
    });
  });
});

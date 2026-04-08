import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  deleteAdminConfig,
  fetchAdminSettingsSections,
  fetchAdminAPILimitsSettings,
  sendAdminMailDeliveryTest,
  upsertAdminConfig,
} from "../api";
import { AdminSettingsPage } from "./settings-page";

vi.mock("../api", () => ({
  deleteAdminConfig: vi.fn(),
  fetchAdminSettingsSections: vi.fn(),
  fetchAdminAPILimitsSettings: vi.fn(),
  sendAdminMailDeliveryTest: vi.fn(),
  upsertAdminConfig: vi.fn(),
}));

describe("AdminSettingsPage", () => {
  beforeEach(() => {
    cleanup();
    vi.clearAllMocks();

    vi.mocked(fetchAdminSettingsSections).mockResolvedValue([
      {
        key: "site",
        title: "站点信息",
        description: "品牌名、联系邮箱、默认语言与时区。",
        items: [
          {
            key: "site.identity",
            value: {
              siteName: "Shiro Email",
              slogan: "Enterprise temporary mail platform",
              supportEmail: "ops@shiro.local",
              appBaseUrl: "https://mail.example.com",
              defaultLanguage: "zh-CN",
              defaultTimeZone: "Asia/Shanghai",
            },
            updatedBy: 3,
            updatedAt: "2026-04-05T10:00:00Z",
          },
        ],
      },
      {
        key: "auth",
        title: "认证与注册",
        description: "注册开放策略、密码规则、会话策略。",
        items: [
          {
            key: "auth.registration_policy",
            value: {
              registrationMode: "public",
              allowRegistration: true,
              requireEmailVerification: false,
              inviteOnly: false,
            },
            updatedBy: 3,
            updatedAt: "2026-04-05T10:00:00Z",
          },
          {
            key: "auth.password_policy",
            value: {
              minLength: 8,
              requireUppercase: true,
              requireNumber: true,
              requireSpecial: false,
              passwordResetable: true,
            },
            updatedBy: 3,
            updatedAt: "2026-04-05T10:00:00Z",
          },
          {
            key: "auth.session_policy",
            value: {
              accessTokenMinutes: 60,
              refreshTokenDays: 7,
              allowMultiSession: true,
              enableMFA: false,
              lockoutThreshold: 5,
              lockoutDurationMinutes: 30,
            },
            updatedBy: 3,
            updatedAt: "2026-04-05T10:00:00Z",
          },
        ],
      },
      {
        key: "oauth",
        title: "OAuth / OIDC",
        description: "第三方登录展示顺序与 provider 凭据。",
        items: [
          {
            key: "auth.oauth.display",
            value: {
              showOnLogin: true,
              providerOrder: ["google", "github"],
              autoLinkByEmail: true,
            },
            updatedBy: 3,
            updatedAt: "2026-04-05T10:00:00Z",
          },
          {
            key: "auth.oauth.providers.google",
            value: {
              enabled: true,
              clientId: "google-client",
              clientSecret: "google-secret",
              redirectUrl: "http://localhost/oauth/google",
              scopes: ["openid", "email"],
              allowAutoRegister: true,
              allowLinkExisting: true,
              overwriteProfile: false,
              displayName: "Google",
            },
            updatedBy: 3,
            updatedAt: "2026-04-05T10:00:00Z",
          },
          {
            key: "auth.oauth.providers.github",
            value: {
              enabled: false,
              clientId: "",
              clientSecret: "",
              redirectUrl: "",
              scopes: ["read:user"],
              allowAutoRegister: true,
              allowLinkExisting: true,
              overwriteProfile: false,
              displayName: "GitHub",
            },
            updatedBy: 3,
            updatedAt: "2026-04-05T10:00:00Z",
          },
          {
            key: "auth.oauth.providers.microsoft",
            value: {
              enabled: false,
              clientId: "",
              clientSecret: "",
              redirectUrl: "",
              scopes: ["openid"],
              allowAutoRegister: true,
              allowLinkExisting: true,
              overwriteProfile: false,
              displayName: "Microsoft",
            },
            updatedBy: 3,
            updatedAt: "2026-04-05T10:00:00Z",
          },
        ],
      },
      {
        key: "mail",
        title: "收件与 SMTP",
        description: "SMTP 监听与收件策略。",
        items: [
          {
            key: "mail.smtp",
            value: {
              enabled: true,
              listenAddr: ":2525",
              hostname: "mail.shiro.local",
              dkimCnameTarget: "shiro._domainkey.shiro.local",
              maxMessageBytes: 10485760,
            },
            updatedBy: 3,
            updatedAt: "2026-04-05T10:00:00Z",
          },
          {
            key: "mail.delivery",
            value: {
              enabled: true,
              host: "smtp.example.com",
              port: 587,
              username: "sender@example.com",
              password: "secret",
              fromAddress: "sender@example.com",
              fromName: "Shiro Email",
            },
            updatedBy: 3,
            updatedAt: "2026-04-05T10:00:00Z",
          },
          {
            key: "mail.inbound_policy",
            value: {
              allowCatchAll: false,
              requireExistingMailbox: true,
              retainRawDays: 30,
              maxAttachmentSizeMB: 15,
              rejectExecutableFiles: true,
              enableSpamScanningPreview: false,
            },
            updatedBy: 3,
            updatedAt: "2026-04-05T10:00:00Z",
          },
        ],
      },
      {
        key: "api",
        title: "API 设置",
        description: "API 速率限制、鉴权身份桶与严格 IP 限流策略。",
        items: [
          {
            key: "api.limits",
            value: {
              enabled: true,
              identityMode: "bearer_or_ip",
              anonymousRPM: 120,
              authenticatedRPM: 600,
              authRPM: 10,
              loginRPM: 10,
              registerRPM: 10,
              refreshRPM: 30,
              forgotPasswordRPM: 10,
              resetPasswordRPM: 10,
              emailVerificationResendRPM: 10,
              emailVerificationConfirmRPM: 30,
              oauthStartRPM: 20,
              oauthCallbackRPM: 20,
              login2faVerifyRPM: 20,
              mailboxWriteRPM: 1200,
              strictIpEnabled: true,
              strictIpRPM: 1800,
            },
            updatedBy: 3,
            updatedAt: "2026-04-05T10:00:00Z",
          },
        ],
      },
      {
        key: "domain",
        title: "域名平台策略",
        description: "公开域发布审核等平台级域名策略。",
        items: [
          {
            key: "domain.public_pool_policy",
            value: {
              requiresReview: true,
            },
            updatedBy: 3,
            updatedAt: "2026-04-05T10:00:00Z",
          },
        ],
      },
    ]);
    vi.mocked(fetchAdminAPILimitsSettings).mockResolvedValue({
      enabled: true,
      identityMode: "bearer_or_ip",
      anonymousRPM: 120,
      authenticatedRPM: 600,
      authRPM: 10,
      loginRPM: 10,
      registerRPM: 10,
      refreshRPM: 30,
      forgotPasswordRPM: 10,
      resetPasswordRPM: 10,
      emailVerificationResendRPM: 10,
      emailVerificationConfirmRPM: 30,
      oauthStartRPM: 20,
      oauthCallbackRPM: 20,
      login2faVerifyRPM: 20,
      mailboxWriteRPM: 1200,
      strictIpEnabled: true,
      strictIpRPM: 1800,
    });

    vi.mocked(upsertAdminConfig).mockImplementation(async (key, value) => ({
      key,
      value,
      updatedBy: 3,
      updatedAt: "2026-04-05T10:05:00Z",
    }));
    vi.mocked(deleteAdminConfig).mockResolvedValue(undefined);
    vi.mocked(sendAdminMailDeliveryTest).mockResolvedValue({
      status: "ok",
      recipient: "sender@example.com",
    });
  });

  it("renders structured settings sections from admin settings api", async () => {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <AdminSettingsPage />
      </QueryClientProvider>,
    );

    expect(
      await screen.findByRole("textbox", { name: "站点名称" }),
    ).toHaveValue("Shiro Email");
    await waitFor(() => {
      expect(screen.getByRole("textbox", { name: "站点地址" })).toHaveValue(
        "https://mail.example.com",
      );
    });
    expect(screen.getByRole("tab", { name: "OAuth 设置" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "用户设置" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "API 设置" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "其他设置" })).toBeInTheDocument();
    expect(vi.mocked(upsertAdminConfig)).not.toHaveBeenCalled();
  });

  it("derives smtp defaults from the configured site domain", async () => {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <AdminSettingsPage />
      </QueryClientProvider>,
    );

    const otherTab = await screen.findByRole("tab", { name: "其他设置" });
    otherTab.focus();
    fireEvent.keyDown(otherTab, { key: "Enter", code: "Enter" });

    expect(
      await screen.findByRole("textbox", { name: "SMTP Hostname" }),
    ).toHaveValue("smtp.example.com");
    expect(
      screen.getByRole("textbox", { name: "DKIM CNAME Target" }),
    ).toHaveValue("shiro._domainkey.example.com");
  });

  it("sends a mail delivery test from the delivery settings panel", async () => {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <AdminSettingsPage />
      </QueryClientProvider>,
    );

    const otherTab = await screen.findByRole("tab", { name: "其他设置" });
    otherTab.focus();
    fireEvent.keyDown(otherTab, { key: "Enter", code: "Enter" });
    await screen.findByText("账户邮件发信");
    fireEvent.click(screen.getByRole("button", { name: "发送测试邮件" }));

    await waitFor(() => {
      expect(vi.mocked(sendAdminMailDeliveryTest)).toHaveBeenCalledWith({
        to: "sender@example.com",
      });
    });
    expect(
      await screen.findByText("最近一次 SMTP 测试"),
    ).toBeInTheDocument();
    expect(screen.getByText("Success")).toBeInTheDocument();
  });

  it("renders structured smtp diagnostics when the test request fails", async () => {
    const diagnosticError = {
      isAxiosError: true,
      response: {
        data: {
          message: "mail delivery TLS handshake failed: server does not advertise STARTTLS",
          stage: "tls",
          code: "starttls_unavailable",
          hint: "The server does not advertise STARTTLS. Switch to Plain SMTP / SMTPS, or enable STARTTLS on the server.",
          retryable: false,
        },
      },
    };
    vi.mocked(sendAdminMailDeliveryTest).mockRejectedValueOnce(diagnosticError);

    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <AdminSettingsPage />
      </QueryClientProvider>,
    );

    const otherTab = await screen.findByRole("tab", { name: "其他设置" });
    otherTab.focus();
    fireEvent.keyDown(otherTab, { key: "Enter", code: "Enter" });
    await screen.findByText("账户邮件发信");
    fireEvent.click(screen.getByRole("button", { name: "发送测试邮件" }));

    expect(await screen.findByText("Failed")).toBeInTheDocument();
    expect(screen.getByText("starttls_unavailable")).toBeInTheDocument();
    expect(screen.getByText("stage: tls")).toBeInTheDocument();
    expect(screen.getAllByText(/The server does not advertise STARTTLS/i).length).toBeGreaterThan(0);
  });

  it("deletes stale oauth provider configs when saving removed providers", async () => {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <AdminSettingsPage />
      </QueryClientProvider>,
    );

    const oauthTab = await screen.findByRole("tab", { name: "OAuth 设置" });
    oauthTab.focus();
    fireEvent.keyDown(oauthTab, { key: "Enter", code: "Enter" });

    await screen.findByText("OAuth 应用");
    const githubCard = screen.getByText("GitHub").closest("div.rounded-xl");
    expect(githubCard).not.toBeNull();
    fireEvent.click(
      within(githubCard as HTMLElement).getByRole("button", { name: "删除" }),
    );
    fireEvent.click(screen.getByRole("button", { name: "保存设置" }));

    await waitFor(() => {
      expect(vi.mocked(deleteAdminConfig)).toHaveBeenCalledWith(
        "auth.oauth.providers.github",
      );
    });
  });

  it("saves api limits settings from the dedicated api tab", async () => {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <AdminSettingsPage />
      </QueryClientProvider>,
    );

    const apiTab = await screen.findByRole("tab", { name: "API 设置" });
    apiTab.focus();
    fireEvent.keyDown(apiTab, { key: "Enter", code: "Enter" });

    const strictIpInput = await screen.findByRole("spinbutton", {
      name: "严格 IP RPM",
    });
    fireEvent.change(strictIpInput, { target: { value: "2400" } });
    fireEvent.click(screen.getByRole("button", { name: "保存设置" }));

    await waitFor(() => {
      expect(vi.mocked(upsertAdminConfig)).toHaveBeenCalledWith("api.limits", {
        enabled: true,
        identityMode: "bearer_or_ip",
        anonymousRPM: 120,
        authenticatedRPM: 600,
        authRPM: 10,
        loginRPM: 10,
        registerRPM: 10,
        refreshRPM: 30,
        forgotPasswordRPM: 10,
        resetPasswordRPM: 10,
        emailVerificationResendRPM: 10,
        emailVerificationConfirmRPM: 30,
        oauthStartRPM: 20,
        oauthCallbackRPM: 20,
        login2faVerifyRPM: 20,
        mailboxWriteRPM: 1200,
        strictIpEnabled: true,
        strictIpRPM: 2400,
      });
    });
  });

  it("shows current effective api limit summary in api tab", async () => {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <AdminSettingsPage />
      </QueryClientProvider>,
    );

    const apiTab = await screen.findByRole("tab", { name: "API 设置" });
    apiTab.focus();
    fireEvent.keyDown(apiTab, { key: "Enter", code: "Enter" });

    expect(await screen.findByText("Rate limit enabled")).toBeInTheDocument();
    expect(screen.getByText("Bearer / IP mixed")).toBeInTheDocument();
    expect(screen.getByText("120 / 600")).toBeInTheDocument();
    expect(screen.getByText("1800 RPM")).toBeInTheDocument();
  });
});

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useAuthStore } from "@/lib/auth-store";
import {
  applyDomainProviderChangeSet,
  createDomain,
  createDomainProvider,
  deleteDomainProvider,
  fetchDomainProviderChangeSets,
  fetchDomainProviderRecords,
  fetchDomainProviderVerifications,
  fetchDomainProviderZones,
  fetchDomainProviders,
  fetchDomains,
  generateSubdomains,
  previewDomainProviderChangeSet,
  updateDomainProvider,
  validateDomainProvider,
} from "../api";
import { UserDnsPage } from "./dns-page";

vi.mock("../api", () => ({
  applyDomainProviderChangeSet: vi.fn(),
  createDomain: vi.fn(),
  createDomainProvider: vi.fn(),
  deleteDomainProvider: vi.fn(),
  fetchDomainProviderChangeSets: vi.fn(),
  fetchDomainProviderRecords: vi.fn(),
  fetchDomainProviderVerifications: vi.fn(),
  fetchDomainProviderZones: vi.fn(),
  fetchDomainProviders: vi.fn(),
  fetchDomains: vi.fn(),
  generateSubdomains: vi.fn(),
  previewDomainProviderChangeSet: vi.fn(),
  updateDomainProvider: vi.fn(),
  validateDomainProvider: vi.fn(),
}));

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  return render(
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>
        <UserDnsPage />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe("UserDnsPage provider form", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useAuthStore.setState({
      accessToken: "token",
      refreshToken: "refresh",
      user: {
        userId: 7,
        username: "tester",
        roles: ["user"],
      },
    });

    vi.mocked(fetchDomains).mockResolvedValue([]);
    vi.mocked(fetchDomainProviders).mockResolvedValue([]);
    vi.mocked(fetchDomainProviderZones).mockResolvedValue([]);
    vi.mocked(fetchDomainProviderRecords).mockResolvedValue([]);
    vi.mocked(fetchDomainProviderChangeSets).mockResolvedValue([]);
    vi.mocked(fetchDomainProviderVerifications).mockResolvedValue([]);
    vi.mocked(previewDomainProviderChangeSet).mockResolvedValue({} as never);
    vi.mocked(applyDomainProviderChangeSet).mockResolvedValue({} as never);
    vi.mocked(createDomainProvider).mockResolvedValue({} as never);
    vi.mocked(validateDomainProvider).mockResolvedValue({} as never);
    vi.mocked(deleteDomainProvider).mockResolvedValue({ ok: true });
    vi.mocked(createDomain).mockResolvedValue({} as never);
    vi.mocked(generateSubdomains).mockResolvedValue([]);
    vi.mocked(updateDomainProvider).mockResolvedValue({} as never);
  });

  it("switches credential fields when auth mode changes", async () => {
    renderPage();

    fireEvent.click(await screen.findByRole("button", { name: "新增 Provider" }));
    const dialog = await screen.findByRole("dialog", { name: "新增 Provider" });
    const dialogQueries = within(dialog);

    expect(dialogQueries.getByLabelText("API Token")).toBeInTheDocument();
    expect(dialogQueries.queryByLabelText("Account Email")).not.toBeInTheDocument();
    expect(dialogQueries.queryByLabelText("Global API Key")).not.toBeInTheDocument();

    fireEvent.click(dialogQueries.getByRole("combobox", { name: "鉴权方式" }));
    fireEvent.click(await screen.findByRole("option", { name: "Global API Key + Email" }));

    expect(dialogQueries.queryByLabelText("API Token")).not.toBeInTheDocument();
    expect(dialogQueries.getByLabelText("Account Email")).toBeInTheDocument();
    expect(dialogQueries.getByLabelText("Global API Key")).toBeInTheDocument();

    fireEvent.click(dialogQueries.getByRole("combobox", { name: "Provider" }));
    fireEvent.click(await screen.findByRole("option", { name: "Spaceship" }));

    expect(dialogQueries.queryByLabelText("Account Email")).not.toBeInTheDocument();
    expect(dialogQueries.queryByLabelText("Global API Key")).not.toBeInTheDocument();
    expect(dialogQueries.queryByLabelText("API Token")).not.toBeInTheDocument();
    expect(dialogQueries.getByLabelText("API Key")).toBeInTheDocument();
    expect(dialogQueries.getByLabelText("API Secret")).toBeInTheDocument();
  });

  it("submits only the active auth fields for global api key mode", async () => {
    renderPage();

    fireEvent.click(await screen.findByRole("button", { name: "新增 Provider" }));
    const dialog = await screen.findByRole("dialog", { name: "新增 Provider" });
    const dialogQueries = within(dialog);

    expect(dialogQueries.queryByLabelText("能力")).not.toBeInTheDocument();
    expect(dialogQueries.getByLabelText("权限")).toBeInTheDocument();

    fireEvent.change(dialogQueries.getByPlaceholderText("例如 My Cloudflare"), {
      target: { value: "CF Global" },
    });
    fireEvent.click(dialogQueries.getByRole("combobox", { name: "鉴权方式" }));
    fireEvent.click(await screen.findByRole("option", { name: "Global API Key + Email" }));
    const permissionCombobox = dialogQueries.getByRole("combobox", { name: "权限" });
    fireEvent.click(permissionCombobox);
    fireEvent.click(await screen.findByRole("option", { name: "DNS 读取" }));

    fireEvent.change(dialogQueries.getByLabelText("Account Email"), {
      target: { value: " user@example.com " },
    });
    fireEvent.change(dialogQueries.getByLabelText("Global API Key"), {
      target: { value: " global-key " },
    });

    const submitButton = screen
      .getAllByText("创建 Provider")
      .map((node) => node.closest("button"))
      .find((button): button is HTMLButtonElement => !!button && !button.disabled);

    expect(submitButton).toBeDefined();
    fireEvent.click(submitButton!);

    await waitFor(() => {
      expect(vi.mocked(createDomainProvider)).toHaveBeenCalledTimes(1);
      expect(vi.mocked(createDomainProvider).mock.calls[0]?.[0]).toEqual({
        provider: "cloudflare",
        displayName: "CF Global",
        authType: "api_key",
        credentials: {
          apiToken: "",
          apiEmail: "user@example.com",
          apiKey: "global-key",
          apiSecret: "",
        },
        status: "pending",
        capabilities: ["zones.read", "dns.write", "dns.read"],
      });
    });
  });

  it("prefills provider dialog for editing and allows saving without replacing credentials", async () => {
    vi.mocked(fetchDomainProviders).mockResolvedValue([
      {
        id: 1,
        provider: "cloudflare",
        ownerType: "user",
        ownerUserId: 7,
        displayName: "My Cloudflare",
        authType: "api_token",
        hasSecret: true,
        status: "healthy",
        capabilities: ["zones.read", "dns.read", "dns.write"],
        createdAt: "2026-04-05T08:00:00Z",
        updatedAt: "2026-04-05T08:00:00Z",
      },
    ]);

    renderPage();

    fireEvent.click(await screen.findByRole("button", { name: "My Cloudflare 编辑" }));

    const dialog = await screen.findByRole("dialog", { name: "编辑 Provider" });
    const dialogQueries = within(dialog);

    expect(dialogQueries.getByDisplayValue("My Cloudflare")).toBeInTheDocument();
    fireEvent.change(dialogQueries.getByPlaceholderText("例如 My Cloudflare"), {
      target: { value: "My Cloudflare Updated" },
    });

    const submitButton = screen
      .getAllByText("保存 Provider")
      .map((node) => node.closest("button"))
      .find((button): button is HTMLButtonElement => !!button && !button.disabled);

    expect(submitButton).toBeDefined();
    fireEvent.click(submitButton!);

    await waitFor(() => {
      expect(vi.mocked(updateDomainProvider)).toHaveBeenCalledTimes(1);
      expect(vi.mocked(updateDomainProvider).mock.calls[0]?.[0]).toBe(1);
      expect(vi.mocked(updateDomainProvider).mock.calls[0]?.[1]).toMatchObject({
        provider: "cloudflare",
        displayName: "My Cloudflare Updated",
        authType: "api_token",
        credentials: {
          apiToken: "",
          apiEmail: "",
          apiKey: "",
          apiSecret: "",
        },
      });
    });
  });

  it("locks provider and auth type when the provider is already bound to domains", async () => {
    vi.mocked(fetchDomains).mockResolvedValue([
      {
        id: 11,
        domain: "bound.example.com",
        status: "active",
        ownerUserId: 7,
        visibility: "private",
        publicationStatus: "draft",
        verificationScore: 100,
        healthStatus: "healthy",
        providerAccountId: 1,
        isDefault: false,
        weight: 100,
        rootDomain: "example.com",
        parentDomain: "example.com",
        level: 1,
        kind: "subdomain",
      },
    ] as never);
    vi.mocked(fetchDomainProviders).mockResolvedValue([
      {
        id: 1,
        provider: "cloudflare",
        ownerType: "user",
        ownerUserId: 7,
        displayName: "Bound Cloudflare",
        authType: "api_token",
        hasSecret: true,
        status: "healthy",
        capabilities: ["zones.read", "dns.read", "dns.write"],
        createdAt: "2026-04-05T08:00:00Z",
        updatedAt: "2026-04-05T08:00:00Z",
      },
    ]);

    renderPage();

    fireEvent.click(await screen.findByRole("button", { name: "Bound Cloudflare 编辑" }));

    const dialog = await screen.findByRole("dialog", { name: "编辑 Provider" });
    const dialogQueries = within(dialog);

    expect(dialogQueries.getByRole("combobox", { name: "Provider" })).toBeDisabled();
    expect(dialogQueries.getByRole("combobox", { name: "鉴权方式" })).toBeDisabled();
    expect(dialogQueries.getByText("当前 Provider 已绑定域名，可继续更新显示名称、凭据、状态和权限，但不能改服务商类型或鉴权方式。")).toBeInTheDocument();
  });
});

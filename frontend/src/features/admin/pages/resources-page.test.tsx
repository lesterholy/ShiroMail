import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  fetchAdminAudit,
  fetchAdminConfigs,
  fetchAdminDomainProviders,
  fetchAdminJobs,
} from "../api";
import { AdminResourcesPage } from "./resources-page";

vi.mock("../api", () => ({
  fetchAdminDomainProviders: vi.fn(),
  fetchAdminConfigs: vi.fn(),
  fetchAdminJobs: vi.fn(),
  fetchAdminAudit: vi.fn(),
}));

describe("AdminResourcesPage", () => {
  beforeEach(() => {
    cleanup();
    vi.clearAllMocks();

    vi.mocked(fetchAdminDomainProviders).mockResolvedValue([
      {
        id: 1,
        provider: "cloudflare",
        ownerType: "platform",
        displayName: "Primary Cloudflare",
        authType: "api_token",
        hasSecret: true,
        status: "healthy",
        capabilities: ["zones.read", "dns.write"],
        createdAt: "2026-04-03T10:00:00Z",
        updatedAt: "2026-04-03T10:05:00Z",
      },
    ]);
    vi.mocked(fetchAdminConfigs).mockResolvedValue([
      {
        key: "platform",
        value: { brand: "Shiro Email" },
        updatedBy: 3,
        updatedAt: "2026-04-03T10:10:00Z",
      },
    ]);
    vi.mocked(fetchAdminJobs).mockResolvedValue([
      {
        id: 1,
        jobType: "mail_ingest_listener",
        status: "ok",
        errorMessage: "",
        createdAt: "2026-04-03T10:15:00Z",
      },
    ]);
    vi.mocked(fetchAdminAudit).mockResolvedValue([
      {
        id: 1,
        actorUserId: 3,
        action: "admin.config.upsert",
        resourceType: "config",
        resourceId: "platform",
        detail: { brand: "Shiro Email" },
        createdAt: "2026-04-03T10:20:00Z",
      },
    ]);
  });

  it("renders real resource inventory sections for admins", async () => {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <AdminResourcesPage />
      </QueryClientProvider>,
    );

    expect(await screen.findAllByText("Provider 账号")).toHaveLength(2);
    expect(await screen.findAllByText("配置注册表")).toHaveLength(2);
    expect(await screen.findAllByText("任务队列")).toHaveLength(2);
    expect(await screen.findByText("DNS Provider 账号")).toBeInTheDocument();
    expect(await screen.findByText("系统配置项")).toBeInTheDocument();
    expect(await screen.findByText("后台任务")).toBeInTheDocument();
    expect(await screen.findByText("Primary Cloudflare")).toBeInTheDocument();
    expect(await screen.findByText("updated by #3")).toBeInTheDocument();
    expect(await screen.findByText("mail_ingest_listener")).toBeInTheDocument();
    expect(await screen.findByText("admin.config.upsert")).toBeInTheDocument();
  });
});

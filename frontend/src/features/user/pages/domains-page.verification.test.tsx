import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useAuthStore } from "@/lib/auth-store";
import { fetchDomainProviders, fetchDomains, verifyDomain } from "../api";
import { UserDomainsPage } from "./domains-page";

vi.mock("../api", () => ({
  fetchDomains: vi.fn(),
  fetchDomainProviders: vi.fn(),
  createDomain: vi.fn(),
  deleteDomain: vi.fn(),
  generateSubdomains: vi.fn(),
  requestDomainPublicPool: vi.fn(),
  updateDomainProviderBinding: vi.fn(),
  verifyDomain: vi.fn(),
  withdrawDomainPublicPool: vi.fn(),
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
        <UserDomainsPage />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe("UserDomainsPage verification", () => {
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

    vi.mocked(fetchDomains).mockResolvedValue([
      {
        id: 1,
        domain: "owned-private.test",
        status: "active",
        ownerUserId: 7,
        visibility: "private",
        publicationStatus: "draft",
        verificationScore: 0,
        healthStatus: "degraded",
        providerAccountId: 11,
        providerDisplayName: "Cloudflare Prod",
        isDefault: false,
        weight: 100,
        rootDomain: "owned-private.test",
        parentDomain: "",
        level: 0,
        kind: "root",
      },
    ]);
    vi.mocked(fetchDomainProviders).mockResolvedValue([]);
    vi.mocked(verifyDomain).mockResolvedValue({
      domain: {
        id: 1,
        domain: "owned-private.test",
        status: "active",
        ownerUserId: 7,
        visibility: "private",
        publicationStatus: "draft",
        verificationScore: 50,
        healthStatus: "degraded",
        providerAccountId: 11,
        providerDisplayName: "Cloudflare Prod",
        isDefault: false,
        weight: 100,
        rootDomain: "owned-private.test",
        parentDomain: "",
        level: 0,
        kind: "root",
      },
      passed: false,
      summary: "DNS 传播验证未通过，请根据缺失或漂移记录继续修复。",
      zoneName: "owned-private.test",
      verifiedCount: 1,
      totalCount: 2,
      profiles: [
        {
          verificationType: "mx",
          status: "drifted",
          summary: "MX 记录仍未对齐",
          expectedRecords: [],
          observedRecords: [],
          repairRecords: [
            {
              type: "MX",
              name: "owned-private.test",
              value: "mx.shiro.email",
              ttl: 120,
              priority: 10,
              proxied: false,
            },
          ],
          lastCheckedAt: "2026-04-05T05:12:00Z",
        },
      ],
    });
  });

  it("shows verification summary and repair records after verify", async () => {
    renderPage();

    fireEvent.click(await screen.findByRole("button", { name: "验证" }));

    await waitFor(() => {
      expect(vi.mocked(verifyDomain)).toHaveBeenCalled();
    });
    expect(vi.mocked(verifyDomain).mock.calls[0]?.[0]).toBe(1);

    expect(await screen.findByText("最近一次验证未通过")).toBeInTheDocument();
    expect(screen.getByText("传播部分通过")).toBeInTheDocument();
    expect(screen.getByText("待修复项")).toBeInTheDocument();
    expect(screen.getByText("1 项")).toBeInTheDocument();
    expect(screen.getByText("MX 记录仍未对齐")).toBeInTheDocument();
    expect(screen.getByText(/mx\.shiro\.email/)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "前往 DNS 配置" })).toHaveAttribute(
      "href",
      "/dashboard/dns?domainId=1&providerId=11",
    );
  });
});

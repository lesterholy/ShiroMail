import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { fetchAdminRules, upsertAdminRule } from "../api";
import { AdminRulesPage } from "./rules-page";

vi.mock("../api", () => ({
  fetchAdminRules: vi.fn(),
  upsertAdminRule: vi.fn(),
}));

describe("AdminRulesPage", () => {
  beforeEach(() => {
    cleanup();
    vi.clearAllMocks();

    vi.mocked(fetchAdminRules).mockResolvedValue([
      {
        id: "default",
        name: "默认保留策略",
        retentionHours: 24,
        autoExtend: false,
        updatedAt: "2026-04-03T10:00:00Z",
      },
      {
        id: "vip",
        name: "高可用邮箱",
        retentionHours: 72,
        autoExtend: true,
        updatedAt: "2026-04-03T11:00:00Z",
      },
    ]);
    vi.mocked(upsertAdminRule).mockImplementation(async (id, input) => ({
      id,
      ...input,
      updatedAt: "2026-04-03T11:05:00Z",
    }));
  });

  it("renders admin rule list from the real rules api", async () => {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <AdminRulesPage />
      </QueryClientProvider>,
    );

    expect(await screen.findByText("默认保留策略")).toBeInTheDocument();
    expect(await screen.findByText("高可用邮箱")).toBeInTheDocument();
    expect(await screen.findByText("72h")).toBeInTheDocument();
  });

  it("saves edited rule payloads through admin rule api", async () => {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <AdminRulesPage />
      </QueryClientProvider>,
    );

    fireEvent.click(await screen.findByRole("button", { name: /高可用邮箱/ }));
    fireEvent.change(screen.getByPlaceholderText("规则名称"), {
      target: { value: "VIP 自动续期策略" },
    });
    fireEvent.change(screen.getByRole("spinbutton"), {
      target: { value: "96" },
    });
    fireEvent.click(screen.getByLabelText("启用自动续期"));
    fireEvent.click(screen.getByRole("button", { name: "保存规则" }));

    await waitFor(() => {
      expect(vi.mocked(upsertAdminRule).mock.calls[0]?.[0]).toBe("vip");
      expect(vi.mocked(upsertAdminRule).mock.calls[0]?.[1]).toEqual({
        name: "VIP 自动续期策略",
        retentionHours: 96,
        autoExtend: false,
      });
    });
  });
});

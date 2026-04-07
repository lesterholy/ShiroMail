import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  createAdminNotice,
  deleteAdminNotice,
  fetchAdminNotices,
  updateAdminNotice,
} from "../api";
import { AdminNoticesPage } from "./notices-page";

vi.mock("../api", () => ({
  fetchAdminNotices: vi.fn(),
  createAdminNotice: vi.fn(),
  updateAdminNotice: vi.fn(),
  deleteAdminNotice: vi.fn(),
}));

describe("AdminNoticesPage", () => {
  beforeEach(() => {
    cleanup();
    vi.clearAllMocks();

    vi.mocked(fetchAdminNotices).mockResolvedValue([
      {
        id: 1,
        title: "平台维护",
        body: "今晚 23:00 进行数据库升级。",
        category: "maintenance",
        level: "warning",
        publishedAt: "2026-04-03T10:00:00Z",
      },
    ]);
    vi.mocked(createAdminNotice).mockResolvedValue({
      id: 2,
      title: "发布新版本",
      body: "Webhook 与 API Key 管理已完成升级。",
      category: "release",
      level: "info",
      publishedAt: "2026-04-03T11:00:00Z",
    });
    vi.mocked(updateAdminNotice).mockResolvedValue({
      id: 1,
      title: "平台维护更新",
      body: "维护窗口已调整到 23:30。",
      category: "maintenance",
      level: "warning",
      publishedAt: "2026-04-03T10:00:00Z",
    });
    vi.mocked(deleteAdminNotice).mockResolvedValue({ ok: true });
  });

  it("renders real admin notices", async () => {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <AdminNoticesPage />
      </QueryClientProvider>,
    );

    expect(await screen.findByText("平台维护")).toBeInTheDocument();
    expect(await screen.findByText("今晚 23:00 进行数据库升级。")).toBeInTheDocument();
    expect(await screen.findByText("maintenance")).toBeInTheDocument();
  });

  it("publishes notices through the real create api", async () => {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <AdminNoticesPage />
      </QueryClientProvider>,
    );

    fireEvent.change(await screen.findByPlaceholderText("公告标题"), {
      target: { value: "发布新版本" },
    });
    fireEvent.change(screen.getByPlaceholderText("公告正文"), {
      target: { value: "Webhook 与 API Key 管理已完成升级。" },
    });
    fireEvent.click(screen.getByRole("button", { name: "发布公告" }));

    await waitFor(() => {
      expect(vi.mocked(createAdminNotice).mock.calls[0]?.[0]).toEqual({
        title: "发布新版本",
        body: "Webhook 与 API Key 管理已完成升级。",
        category: "platform",
        level: "info",
      });
    });
  });

  it("updates and deletes notices from the admin list", async () => {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <AdminNoticesPage />
      </QueryClientProvider>,
    );

    fireEvent.click(await screen.findByRole("button", { name: "编辑" }));
    fireEvent.change(await screen.findByDisplayValue("平台维护"), {
      target: { value: "平台维护更新" },
    });
    fireEvent.change(screen.getByDisplayValue("今晚 23:00 进行数据库升级。"), {
      target: { value: "维护窗口已调整到 23:30。" },
    });
    fireEvent.click(screen.getByRole("button", { name: "保存修改" }));

    await waitFor(() => {
      expect(vi.mocked(updateAdminNotice).mock.calls[0]).toEqual([
        1,
        {
          title: "平台维护更新",
          body: "维护窗口已调整到 23:30。",
          category: "maintenance",
          level: "warning",
        },
      ]);
    });

    fireEvent.click(await screen.findByRole("button", { name: "删除" }));
    fireEvent.click(await screen.findByRole("button", { name: "确认删除" }));

    await waitFor(() => {
      expect(vi.mocked(deleteAdminNotice).mock.calls[0]?.[0]).toBe(1);
    });
  });
});

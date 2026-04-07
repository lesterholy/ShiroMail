import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  createAdminDoc,
  deleteAdminDoc,
  fetchAdminDocs,
  updateAdminDoc,
} from "../api";
import { AdminDocsPage } from "./docs-page";

vi.mock("../api", () => ({
  fetchAdminDocs: vi.fn(),
  createAdminDoc: vi.fn(),
  updateAdminDoc: vi.fn(),
  deleteAdminDoc: vi.fn(),
}));

describe("AdminDocsPage", () => {
  beforeEach(() => {
    cleanup();
    vi.clearAllMocks();

    vi.mocked(fetchAdminDocs).mockResolvedValue([
      {
        id: "quick-start",
        title: "快速开始",
        category: "接入指南",
        summary: "5 分钟完成登录、创建邮箱与轮询收件。",
        readTimeMin: 5,
        tags: ["登录", "邮箱", "轮询"],
        createdAt: "2026-04-03T10:00:00Z",
        updatedAt: "2026-04-03T10:00:00Z",
      },
    ]);
    vi.mocked(createAdminDoc).mockResolvedValue({
      id: "webhook-events",
      title: "Webhook 事件",
      category: "开发文档",
      summary: "说明收件、续期、释放邮箱等事件结构。",
      readTimeMin: 6,
      tags: ["Webhook", "事件", "回调"],
      createdAt: "2026-04-03T11:00:00Z",
      updatedAt: "2026-04-03T11:00:00Z",
    });
    vi.mocked(updateAdminDoc).mockResolvedValue({
      id: "quick-start",
      title: "快速开始（更新）",
      category: "接入指南",
      summary: "10 分钟完成登录、创建邮箱与轮询收件。",
      readTimeMin: 10,
      tags: ["登录", "邮箱"],
      createdAt: "2026-04-03T10:00:00Z",
      updatedAt: "2026-04-03T12:00:00Z",
    });
    vi.mocked(deleteAdminDoc).mockResolvedValue({ ok: true });
  });

  it("renders real doc articles for admins instead of audit rows", async () => {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <AdminDocsPage />
      </QueryClientProvider>,
    );

    expect(await screen.findByText("快速开始")).toBeInTheDocument();
    expect(await screen.findByText("接入指南")).toBeInTheDocument();
    expect(await screen.findByText("5 min")).toBeInTheDocument();
    expect(await screen.findByText("5 分钟完成登录、创建邮箱与轮询收件。")).toBeInTheDocument();
  });

  it("creates, updates and deletes docs through admin crud apis", async () => {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <AdminDocsPage />
      </QueryClientProvider>,
    );

    fireEvent.change(await screen.findByPlaceholderText("例如：Webhook 事件"), {
      target: { value: "Webhook 事件" },
    });
    fireEvent.change(screen.getByPlaceholderText("例如：开发文档"), {
      target: { value: "开发文档" },
    });
    fireEvent.change(screen.getByPlaceholderText("使用逗号分隔，如 API, Webhook, 鉴权"), {
      target: { value: "Webhook, 事件, 回调" },
    });
    fireEvent.change(
      screen.getByPlaceholderText("输入文档摘要，前台文档中心会直接展示这里的内容。"),
      {
        target: { value: "说明收件、续期、释放邮箱等事件结构。" },
      },
    );
    fireEvent.click(screen.getByRole("button", { name: "新增文档" }));

    await waitFor(() => {
      expect(vi.mocked(createAdminDoc).mock.calls[0]?.[0]).toEqual({
        title: "Webhook 事件",
        category: "开发文档",
        summary: "说明收件、续期、释放邮箱等事件结构。",
        readTimeMin: 5,
        tags: ["Webhook", "事件", "回调"],
      });
    });

    fireEvent.click(await screen.findByRole("button", { name: "编辑" }));
    const editDialog = await screen.findByRole("dialog", { name: "编辑文档" });
    const dialogQueries = within(editDialog);

    fireEvent.change(dialogQueries.getByDisplayValue("快速开始"), {
      target: { value: "快速开始（更新）" },
    });
    fireEvent.change(dialogQueries.getByDisplayValue("5"), {
      target: { value: "10" },
    });
    fireEvent.change(dialogQueries.getByDisplayValue("登录, 邮箱, 轮询"), {
      target: { value: "登录, 邮箱" },
    });
    fireEvent.change(dialogQueries.getByDisplayValue("5 分钟完成登录、创建邮箱与轮询收件。"), {
      target: { value: "10 分钟完成登录、创建邮箱与轮询收件。" },
    });
    fireEvent.click(dialogQueries.getByRole("button", { name: "保存修改" }));

    await waitFor(() => {
      expect(vi.mocked(updateAdminDoc).mock.calls[0]).toEqual([
        "quick-start",
        {
          title: "快速开始（更新）",
          category: "接入指南",
          summary: "10 分钟完成登录、创建邮箱与轮询收件。",
          readTimeMin: 10,
          tags: ["登录", "邮箱"],
        },
      ]);
    });

    fireEvent.click(await screen.findByRole("button", { name: "删除" }));
    fireEvent.click(await screen.findByRole("button", { name: "确认删除" }));

    await waitFor(() => {
      expect(vi.mocked(deleteAdminDoc).mock.calls[0]?.[0]).toBe("quick-start");
    });
  });
});

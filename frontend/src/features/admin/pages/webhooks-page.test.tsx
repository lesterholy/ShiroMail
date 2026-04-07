import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  createAdminWebhook,
  fetchAdminWebhooks,
  toggleAdminWebhook,
  updateAdminWebhook,
} from "../api";
import { AdminWebhooksPage } from "./webhooks-page";

vi.mock("../api", () => ({
  fetchAdminWebhooks: vi.fn(),
  createAdminWebhook: vi.fn(),
  updateAdminWebhook: vi.fn(),
  toggleAdminWebhook: vi.fn(),
}));

describe("AdminWebhooksPage", () => {
  beforeEach(() => {
    cleanup();
    vi.clearAllMocks();

    vi.mocked(fetchAdminWebhooks).mockResolvedValue([
      {
        id: 1,
        userId: 7,
        name: "primary",
        targetUrl: "https://sandbox.local/webhooks/primary",
        secretPreview: "whsec_123",
        events: ["message.received"],
        enabled: true,
        createdAt: "2026-04-03T10:00:00Z",
        updatedAt: "2026-04-03T10:05:00Z",
      },
    ]);

    vi.mocked(createAdminWebhook).mockResolvedValue({
      id: 2,
      userId: 9,
      name: "ops-events",
      targetUrl: "https://sandbox.local/webhooks/ops-events",
      secretPreview: "whsec_456",
      events: ["message.received", "mailbox.released"],
      enabled: true,
      createdAt: "2026-04-03T10:00:00Z",
      updatedAt: "2026-04-03T10:10:00Z",
    });
    vi.mocked(updateAdminWebhook).mockResolvedValue({
      id: 1,
      userId: 7,
      name: "primary-updated",
      targetUrl: "https://sandbox.local/webhooks/primary-updated",
      secretPreview: "whsec_123",
      events: ["message.received", "mailbox.released"],
      enabled: true,
      createdAt: "2026-04-03T10:00:00Z",
      updatedAt: "2026-04-03T10:12:00Z",
    });
    vi.mocked(toggleAdminWebhook).mockResolvedValue({
      id: 1,
      userId: 7,
      name: "primary",
      targetUrl: "https://sandbox.local/webhooks/primary",
      secretPreview: "whsec_123",
      events: ["message.received"],
      enabled: false,
      createdAt: "2026-04-03T10:00:00Z",
      updatedAt: "2026-04-03T10:12:00Z",
    });
  });

  it("renders admin webhook rows with target user context", async () => {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <AdminWebhooksPage />
      </QueryClientProvider>,
    );

    expect(await screen.findByText("primary")).toBeInTheDocument();
    expect(await screen.findByText("https://sandbox.local/webhooks/primary")).toBeInTheDocument();
    expect(await screen.findByText("user #7")).toBeInTheDocument();
  });

  it("creates admin webhooks inside a dialog", async () => {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <AdminWebhooksPage />
      </QueryClientProvider>,
    );

    expect(screen.queryByPlaceholderText("输入用户 ID")).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "新增 Webhook" }));

    const dialog = await screen.findByRole("dialog", { name: "新增 Webhook" });
    const dialogQueries = within(dialog);

    fireEvent.change(dialogQueries.getByPlaceholderText("输入用户 ID"), {
      target: { value: "9" },
    });
    fireEvent.change(dialogQueries.getByPlaceholderText("Webhook 名称"), {
      target: { value: "ops-events" },
    });
    fireEvent.change(
      dialogQueries.getByPlaceholderText("https://sandbox.local/webhooks/order"),
      {
        target: { value: "https://sandbox.local/webhooks/ops-events" },
      },
    );
    fireEvent.change(
      dialogQueries.getByPlaceholderText("message.received, mailbox.released"),
      {
        target: { value: "message.received, mailbox.released" },
      },
    );

    fireEvent.click(dialogQueries.getByRole("button", { name: "创建 Webhook" }));

    await waitFor(() => {
      expect(vi.mocked(createAdminWebhook).mock.calls[0]?.[0]).toEqual({
        userId: 9,
        name: "ops-events",
        targetUrl: "https://sandbox.local/webhooks/ops-events",
        events: ["message.received", "mailbox.released"],
      });
    });
  });

  it("reuses the dialog for editing admin webhooks", async () => {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <AdminWebhooksPage />
      </QueryClientProvider>,
    );

    fireEvent.click(await screen.findByRole("button", { name: "编辑" }));

    const dialog = await screen.findByRole("dialog", { name: "编辑 Webhook" });
    const dialogQueries = within(dialog);

    expect(dialogQueries.getByPlaceholderText("输入用户 ID")).toBeDisabled();
    fireEvent.change(dialogQueries.getByPlaceholderText("Webhook 名称"), {
      target: { value: "primary-updated" },
    });
    fireEvent.change(
      dialogQueries.getByPlaceholderText("https://sandbox.local/webhooks/order"),
      {
        target: { value: "https://sandbox.local/webhooks/primary-updated" },
      },
    );
    fireEvent.change(
      dialogQueries.getByPlaceholderText("message.received, mailbox.released"),
      {
        target: { value: "message.received, mailbox.released" },
      },
    );
    fireEvent.click(dialogQueries.getByRole("button", { name: "保存修改" }));

    await waitFor(() => {
      expect(vi.mocked(updateAdminWebhook).mock.calls[0]?.[0]).toBe(1);
      expect(vi.mocked(updateAdminWebhook).mock.calls[0]?.[1]).toEqual({
        name: "primary-updated",
        targetUrl: "https://sandbox.local/webhooks/primary-updated",
        events: ["message.received", "mailbox.released"],
      });
    });
  });

  it("toggles admin webhooks from the list", async () => {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <AdminWebhooksPage />
      </QueryClientProvider>,
    );

    fireEvent.click(await screen.findByRole("button", { name: "停用" }));
    fireEvent.click(await screen.findByRole("button", { name: "确认停用" }));

    await waitFor(() => {
      expect(vi.mocked(toggleAdminWebhook).mock.calls[0]?.[0]).toBe(1);
      expect(vi.mocked(toggleAdminWebhook).mock.calls[0]?.[1]).toBe(false);
    });
  });

  it("shows a dialog error when webhook save fails", async () => {
    vi.mocked(createAdminWebhook).mockRejectedValueOnce(new Error("target url is invalid"));

    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <AdminWebhooksPage />
      </QueryClientProvider>,
    );

    fireEvent.click(screen.getByRole("button", { name: "新增 Webhook" }));

    const dialog = await screen.findByRole("dialog", { name: "新增 Webhook" });
    const dialogQueries = within(dialog);

    fireEvent.change(dialogQueries.getByPlaceholderText("输入用户 ID"), {
      target: { value: "9" },
    });

    fireEvent.click(dialogQueries.getByRole("button", { name: "创建 Webhook" }));

    expect(await dialogQueries.findByText("target url is invalid")).toBeInTheDocument();
  });
});

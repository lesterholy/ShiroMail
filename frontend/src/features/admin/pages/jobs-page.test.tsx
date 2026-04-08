import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  fetchAdminInboundSpool,
  fetchAdminJobs,
  fetchAdminSMTPMetrics,
  retryAdminInboundSpoolItem,
} from "../api";
import { AdminJobsPage } from "./jobs-page";

vi.mock("../api", () => ({
  fetchAdminJobs: vi.fn(),
  fetchAdminInboundSpool: vi.fn(),
  fetchAdminSMTPMetrics: vi.fn(),
  retryAdminInboundSpoolItem: vi.fn(),
}));

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  render(
    <QueryClientProvider client={queryClient}>
      <AdminJobsPage />
    </QueryClientProvider>,
  );
}

describe("AdminJobsPage", () => {
  beforeEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders job history, spool summary, and failure reasons", async () => {
    vi.mocked(fetchAdminJobs).mockResolvedValue([
      {
        id: 1,
        jobType: "inbound_spool",
        status: "failed",
        errorMessage: "temporary parse failure",
        diagnostic: {
          code: "temporary_parse_failure",
          title: "Temporary Parse Failure",
          description:
            "The worker failed while parsing MIME content or message structure. This is often retryable after transient input or runtime issues clear.",
          retryable: true,
        },
        createdAt: "2026-04-08T10:00:00Z",
      },
    ]);
    vi.mocked(fetchAdminSMTPMetrics).mockResolvedValue({
      sessionsStarted: 4,
      recipientsAccepted: 6,
      bytesReceived: 2048,
      accepted: { spool: 3, direct: 1 },
      rejected: { attachment_too_large: 1 },
      spoolProcessed: { completed: 2, failed: 1 },
    });
    vi.mocked(fetchAdminInboundSpool).mockResolvedValue({
      items: [
        {
          id: 8,
          mailFrom: "sender@example.com",
          recipients: ["queued@example.test"],
          targetMailboxIds: [3],
          status: "failed",
          errorMessage: "mailbox not found",
          attemptCount: 3,
          maxAttempts: 3,
          createdAt: "2026-04-08T10:00:00Z",
          updatedAt: "2026-04-08T10:02:00Z",
          nextAttemptAt: "",
          processedAt: undefined,
        },
      ],
      total: 1,
      page: 1,
      pageSize: 10,
      summary: {
        total: 1,
        pending: 0,
        processing: 0,
        completed: 0,
        failed: 1,
      },
      failureReasons: [{ message: "mailbox not found", count: 1 }],
    });

    renderPage();

    expect(await screen.findByText("任务队列")).toBeInTheDocument();
    expect(await screen.findByText("#1 · inbound_spool")).toBeInTheDocument();
    expect(await screen.findByText("Temporary Parse Failure")).toBeInTheDocument();
    expect(await screen.findByText("#8 · sender@example.com")).toBeInTheDocument();
    expect(
      await screen.findAllByText(
        "The message reached spool, but the worker could not match one or more target mailboxes during persistence.",
      ),
    ).toHaveLength(2);
    expect(await screen.findByText("Failed Spool")).toBeInTheDocument();
    expect(await screen.findByText("SMTP 实时指标")).toBeInTheDocument();
    expect(await screen.findByText("Attachment Too Large")).toBeInTheDocument();
    expect(await screen.findByText("Mailbox Not Found")).toBeInTheDocument();
    expect(await screen.findAllByText("Retryable")).not.toHaveLength(0);
    expect(await screen.findByRole("button", { name: /重试/i })).toBeInTheDocument();
  });

  it("retries failed spool item from the admin page", async () => {
    vi.mocked(fetchAdminJobs).mockResolvedValue([]);
    vi.mocked(fetchAdminSMTPMetrics).mockResolvedValue({
      sessionsStarted: 0,
      recipientsAccepted: 0,
      bytesReceived: 0,
      accepted: {},
      rejected: {},
      spoolProcessed: {},
    });
    vi.mocked(fetchAdminInboundSpool).mockResolvedValue({
      items: [
        {
          id: 18,
          mailFrom: "sender@example.com",
          recipients: ["queued@example.test"],
          targetMailboxIds: [3],
          status: "failed",
          errorMessage: "mailbox not found",
          attemptCount: 3,
          maxAttempts: 3,
          createdAt: "2026-04-08T10:00:00Z",
          updatedAt: "2026-04-08T10:02:00Z",
          nextAttemptAt: "",
          processedAt: undefined,
        },
      ],
      total: 1,
      page: 1,
      pageSize: 10,
      summary: {
        total: 1,
        pending: 0,
        processing: 0,
        completed: 0,
        failed: 1,
      },
      failureReasons: [{ message: "mailbox not found", count: 1 }],
    });
    vi.mocked(retryAdminInboundSpoolItem).mockResolvedValue({
      id: 18,
      mailFrom: "sender@example.com",
      recipients: ["queued@example.test"],
      targetMailboxIds: [3],
      status: "pending",
      errorMessage: "",
      attemptCount: 0,
      maxAttempts: 3,
      createdAt: "2026-04-08T10:00:00Z",
      updatedAt: "2026-04-08T10:03:00Z",
      nextAttemptAt: "2026-04-08T10:03:00Z",
      processedAt: undefined,
    });

    renderPage();

    fireEvent.click(await screen.findByRole("button", { name: /重试/i }));

    await waitFor(() => {
      expect(retryAdminInboundSpoolItem).toHaveBeenCalled();
    });
    expect(vi.mocked(retryAdminInboundSpoolItem).mock.calls[0]?.[0]).toBe(18);
  });

  it("renders the inbound spool failure mode filter", async () => {
    vi.mocked(fetchAdminJobs).mockResolvedValue([]);
    vi.mocked(fetchAdminSMTPMetrics).mockResolvedValue({
      sessionsStarted: 0,
      recipientsAccepted: 0,
      bytesReceived: 0,
      accepted: {},
      rejected: {},
      rejectedDetails: [],
      spoolProcessed: {},
    });
    vi.mocked(fetchAdminInboundSpool).mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      pageSize: 10,
      summary: {
        total: 0,
        pending: 0,
        processing: 0,
        completed: 0,
        failed: 0,
      },
      failureReasons: [],
    });

    renderPage();

    await screen.findByText("任务队列");

    expect(screen.getByLabelText("Inbound Spool 失败诊断过滤")).toBeInTheDocument();
    expect(vi.mocked(fetchAdminInboundSpool)).toHaveBeenCalledWith(
      expect.objectContaining({ failureMode: "all" }),
    );
  });
});

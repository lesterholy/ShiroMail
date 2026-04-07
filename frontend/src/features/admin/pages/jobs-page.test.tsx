import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { fetchAdminJobs } from "../api";
import { AdminJobsPage } from "./jobs-page";

vi.mock("../api", () => ({
  fetchAdminJobs: vi.fn(),
}));

describe("AdminJobsPage", () => {
  beforeEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders job rows when records exist", async () => {
    vi.mocked(fetchAdminJobs).mockResolvedValue([
      {
        id: 1,
        jobType: "mail_ingest_listener",
        status: "failed",
        errorMessage: "network timeout",
        createdAt: "2026-04-03T10:00:00Z",
      },
    ]);

    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
      },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <AdminJobsPage />
      </QueryClientProvider>,
    );

    expect(await screen.findByText("mail_ingest_listener")).toBeInTheDocument();
    expect(await screen.findByText("network timeout")).toBeInTheDocument();
    expect(await screen.findByText("failed")).toBeInTheDocument();
  });

  it("shows empty state when no jobs exist", async () => {
    vi.mocked(fetchAdminJobs).mockResolvedValue([]);

    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
      },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <AdminJobsPage />
      </QueryClientProvider>,
    );

    expect(await screen.findByText("暂无任务记录")).toBeInTheDocument();
    expect(await screen.findByText("任务队列当前没有待观察记录。")).toBeInTheDocument();
  });
});

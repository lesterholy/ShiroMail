import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { AccountEmailCard } from "./account-email-card";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, values?: Record<string, string>) => {
      if (key === "account.emailVerificationSent") {
        return `验证码已发送至 ${values?.email ?? ""}`;
      }
      return key;
    },
  }),
}));

describe("AccountEmailCard", () => {
  it("prefills change-email ticket and code from mail action link", () => {
    render(
      <AccountEmailCard
        initialChallenge={{
          status: "verification_required",
          email: "new@example.com",
          verificationTicket: "ticket-1",
          expiresInSeconds: 0,
        }}
        initialCode="123456"
        isConfirmPending={false}
        isRequestPending={false}
        onConfirmChange={vi.fn()}
        onRequestChange={vi.fn()}
        profile={{
          userId: 1,
          username: "tester",
          email: "old@example.com",
          emailVerified: true,
          roles: ["user"],
          displayName: "tester",
          locale: "zh-CN",
          timezone: "Asia/Shanghai",
          autoRefreshSeconds: 30,
          twoFactorEnabled: false,
        }}
      />,
    );

    expect(screen.getByDisplayValue("new@example.com")).toBeInTheDocument();
    expect(screen.getByDisplayValue("123456")).toBeInTheDocument();
    expect(screen.getByText("验证码已发送至 new@example.com")).toBeInTheDocument();
  });
});

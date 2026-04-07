import { cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { AuthSettingsForm } from "./auth-settings-form";
import {
  defaultAuthPasswordSettings,
  defaultAuthRegistrationSettings,
  defaultAuthSessionSettings,
  defaultOAuthDisplaySettings,
  defaultOAuthProviderSettings,
} from "./defaults";
import type {
  OAuthDisplaySettings,
  OAuthProviderSettings,
} from "./types";

function renderForm({
  providers = [],
  oauthDisplay = defaultOAuthDisplaySettings,
  onProvidersChange = vi.fn(),
  onOAuthDisplayChange = vi.fn(),
}: {
  providers?: OAuthProviderSettings[];
  oauthDisplay?: OAuthDisplaySettings;
  onProvidersChange?: (next: OAuthProviderSettings[]) => void;
  onOAuthDisplayChange?: (next: OAuthDisplaySettings) => void;
} = {}) {
  return render(
    <AuthSettingsForm
      registration={defaultAuthRegistrationSettings}
      password={defaultAuthPasswordSettings}
      session={defaultAuthSessionSettings}
      oauthDisplay={oauthDisplay}
      providers={providers}
      onRegistrationChange={vi.fn()}
      onPasswordChange={vi.fn()}
      onSessionChange={vi.fn()}
      onOAuthDisplayChange={onOAuthDisplayChange}
      onProvidersChange={onProvidersChange}
      mode="oauth"
    />,
  );
}

describe("AuthSettingsForm", () => {
  beforeEach(() => {
    cleanup();
    vi.clearAllMocks();
    vi.unstubAllGlobals();
  });

  it("creates a provider from preset and appends it to provider order", () => {
    const onProvidersChange = vi.fn();
    const onOAuthDisplayChange = vi.fn();

    renderForm({ onProvidersChange, onOAuthDisplayChange });

    fireEvent.click(screen.getByRole("button", { name: "添加 OAuth 应用" }));
    fireEvent.click(screen.getByRole("combobox", { name: "预设模板" }));
    fireEvent.click(screen.getByRole("option", { name: "Discord" }));
    fireEvent.change(screen.getByLabelText("OAuth Provider Slug"), {
      target: { value: "discord-sso" },
    });
    fireEvent.change(screen.getByLabelText("OAuth 应用名称"), {
      target: { value: "Discord SSO" },
    });
    fireEvent.click(screen.getByRole("button", { name: "从预设创建" }));

    expect(onProvidersChange).toHaveBeenCalledWith([
      expect.objectContaining({
        slug: "discord-sso",
        displayName: "Discord SSO",
        authorizationUrl: "https://discord.com/oauth2/authorize",
        tokenUrl: "https://discord.com/api/oauth2/token",
        userInfoUrl: "https://discord.com/api/users/@me",
        scopes: ["identify", "email"],
      }),
    ]);
    expect(onOAuthDisplayChange).toHaveBeenCalledWith(
      expect.objectContaining({
        providerOrder: [...defaultOAuthDisplaySettings.providerOrder, "discord-sso"],
      }),
    );
  });

  it("copies callback url and keeps slug read only for existing provider", async () => {
    const provider = {
      ...defaultOAuthProviderSettings("GitHub", "github"),
      redirectUrl: "https://example.com/auth/callback/github",
    };
    const clipboardWriteText = vi.fn().mockResolvedValue(undefined);
    vi.stubGlobal("navigator", {
      clipboard: {
        writeText: clipboardWriteText,
      },
    });

    renderForm({ providers: [provider] });

    const slugInput = screen.getByLabelText("github Provider Slug");
    expect(slugInput).toHaveValue("github");
    expect(slugInput).toHaveAttribute("readonly");

    fireEvent.click(screen.getByRole("button", { name: "复制" }));

    expect(clipboardWriteText).toHaveBeenCalledWith(
      "https://example.com/auth/callback/github",
    );
  });

  it("shows callback preview in create dialog and copies generated url", async () => {
    const clipboardWriteText = vi.fn().mockResolvedValue(undefined);
    vi.stubGlobal("navigator", {
      clipboard: {
        writeText: clipboardWriteText,
      },
    });

    renderForm();

    fireEvent.click(screen.getByRole("button", { name: "添加 OAuth 应用" }));
    fireEvent.change(screen.getByLabelText("OAuth Provider Slug"), {
      target: { value: "gitlab-sso" },
    });

    expect(screen.getByLabelText("新应用回调地址")).toHaveValue(
      "http://localhost:3000/auth/callback/gitlab-sso",
    );

    fireEvent.click(screen.getByRole("button", { name: "复制回调地址" }));

    expect(clipboardWriteText).toHaveBeenCalledWith(
      "http://localhost:3000/auth/callback/gitlab-sso",
    );
  });

  it("uses preset slug for callback preview when custom slug is empty", () => {
    renderForm();

    fireEvent.click(screen.getByRole("button", { name: "添加 OAuth 应用" }));
    fireEvent.click(screen.getByRole("combobox", { name: "预设模板" }));
    fireEvent.click(screen.getByRole("option", { name: "Slack" }));

    expect(screen.getByLabelText("新应用回调地址")).toHaveValue(
      "http://localhost:3000/auth/callback/slack",
    );
  });

  it("disables create action and shows duplicate hint when slug already exists", () => {
    renderForm({
      providers: [defaultOAuthProviderSettings("Slack", "slack")],
    });

    fireEvent.click(screen.getByRole("button", { name: "添加 OAuth 应用" }));
    fireEvent.change(screen.getByLabelText("OAuth Provider Slug"), {
      target: { value: "slack" },
    });

    expect(screen.getByText("该 Provider Slug 已存在，请更换后再创建。")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "创建自定义应用" })).toBeDisabled();
  });

  it("shows selected preset details for quick verification", () => {
    renderForm();

    fireEvent.click(screen.getByRole("button", { name: "添加 OAuth 应用" }));
    fireEvent.click(screen.getByRole("combobox", { name: "预设模板" }));
    fireEvent.click(screen.getByRole("option", { name: "Google" }));

    expect(screen.getByText("预设详情")).toBeInTheDocument();
    expect(screen.getByText("https://accounts.google.com/o/oauth2/v2/auth")).toBeInTheDocument();
    expect(screen.getByText("openid, email, profile")).toBeInTheDocument();
    expect(screen.getByText("PKCE / OAuth 2.1")).toBeInTheDocument();
  });

  it("disables preset option when provider already exists", () => {
    renderForm({
      providers: [defaultOAuthProviderSettings("Google", "google")],
    });

    fireEvent.click(screen.getByRole("button", { name: "添加 OAuth 应用" }));
    fireEvent.click(screen.getByRole("combobox", { name: "预设模板" }));

    const googleOption = screen.getByRole("option", { name: "Google" });
    expect(googleOption).toHaveAttribute("aria-disabled", "true");
    expect(screen.getByText("已接入的预设模板会自动禁用，避免重复创建。")).toBeInTheDocument();
  });

  it("shows configured state summary for connected providers", () => {
    renderForm({
      providers: [
        {
          ...defaultOAuthProviderSettings("GitHub", "github"),
          enabled: true,
          clientId: "client-id",
          clientSecret: "client-secret",
          scopes: ["read:user", "user:email"],
        },
      ],
    });

    expect(screen.getByText("已接入")).toBeInTheDocument();
    expect(screen.getByText("客户端已配置")).toBeInTheDocument();
    expect(screen.getByText("2 个 Scope")).toBeInTheDocument();
  });

  it("removes provider and prunes provider order", () => {
    const onProvidersChange = vi.fn();
    const onOAuthDisplayChange = vi.fn();
    const provider = defaultOAuthProviderSettings("Discord", "discord");

    renderForm({
      providers: [provider],
      oauthDisplay: {
        ...defaultOAuthDisplaySettings,
        providerOrder: ["google", "discord"],
      },
      onProvidersChange,
      onOAuthDisplayChange,
    });

    const providerCard = screen.getByText("Discord").closest("div.rounded-xl");
    expect(providerCard).not.toBeNull();
    fireEvent.click(within(providerCard as HTMLElement).getByRole("button", { name: "删除" }));

    expect(onProvidersChange).toHaveBeenCalledWith([]);
    expect(onOAuthDisplayChange).toHaveBeenCalledWith(
      expect.objectContaining({
        providerOrder: ["google"],
      }),
    );
  });
});

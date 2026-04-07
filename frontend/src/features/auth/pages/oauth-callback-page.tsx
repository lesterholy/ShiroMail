import { useEffect, useMemo, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";
import { completeOAuthLogin } from "../api";
import { composePageTitle, usePageTitle } from "../../../hooks/use-page-title";
import { useSiteName } from "../../../hooks/use-site-name";
import { getDefaultRouteForRoles } from "../../../lib/auth";
import { useAuthStore } from "../../../lib/auth-store";
import { getAPIErrorMessage } from "../../../lib/http";

export function OAuthCallbackPage() {
  const navigate = useNavigate();
  const { provider = "" } = useParams();
  const [searchParams] = useSearchParams();
  const setSession = useAuthStore((state) => state.setSession);
  const [error, setError] = useState<string | null>(null);
  const siteName = useSiteName();

  const code = searchParams.get("code") ?? "";
  const state = searchParams.get("state") ?? "";
  const providerError = searchParams.get("error") ?? "";
  const callbackError = useMemo(() => {
    if (providerError) {
      return null;
    }
    if (!provider) {
      return "缺少 OAuth provider。";
    }
    if (!code || !state) {
      return "OAuth 回调参数不完整。";
    }
    return null;
  }, [code, provider, providerError, state]);

  const description = useMemo(() => {
    if (providerError) {
      return `OAuth provider returned: ${providerError}`;
    }
    if (callbackError) {
      return callbackError;
    }
    if (error) {
      return error;
    }
    return "正在完成 OAuth 登录并跳转到对应工作台...";
  }, [callbackError, error, providerError]);
  usePageTitle(composePageTitle(provider ? `OAuth · ${provider}` : "OAuth", siteName));

  useEffect(() => {
    if (providerError || callbackError) {
      return;
    }

    let cancelled = false;

    void completeOAuthLogin(provider, { code, state })
      .then((session) => {
        if (cancelled) {
          return;
        }
        if (session.kind === "verification_required") {
          navigate(
            `/auth/verify-email?ticket=${encodeURIComponent(session.challenge.verificationTicket)}&email=${encodeURIComponent(session.challenge.email)}`,
            { replace: true },
          );
          return;
        }
        setSession(session.session);
        navigate(getDefaultRouteForRoles(session.session.user.roles), { replace: true });
      })
      .catch((currentError) => {
        if (cancelled) {
          return;
        }
        setError(
          getAPIErrorMessage(
            currentError,
            "OAuth 登录失败，请返回首页重试。",
          ),
        );
      });

    return () => {
      cancelled = true;
    };
  }, [callbackError, code, navigate, provider, providerError, setSession, state]);

  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-6">
      <div className="w-full max-w-md rounded-2xl border border-border/60 bg-card p-6 shadow-sm">
        <p className="text-sm font-medium text-foreground">
          {provider ? `OAuth · ${provider}` : "OAuth"}
        </p>
        <p className="mt-3 text-sm leading-6 text-muted-foreground">
          {description}
        </p>
      </div>
    </div>
  );
}

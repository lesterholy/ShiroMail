import axios from "axios";
import { useAuthStore } from "./auth-store";
import type { AuthResponse } from "./auth";

const API_BASE_URL = String(import.meta.env.VITE_API_BASE_URL ?? "").trim();
const API_URL = API_BASE_URL
  ? `${API_BASE_URL.replace(/\/$/, "")}/api/v1`
  : "/api/v1";

export const http = axios.create({
  baseURL: API_URL,
  timeout: 10000,
});

const authClient = axios.create({
  baseURL: API_URL,
  timeout: 10000,
});

type RetryableRequestConfig = {
  _retry?: boolean;
};

http.interceptors.request.use((config) => {
  const token = useAuthStore.getState().accessToken;
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

let refreshPromise: Promise<string | null> | null = null;

function resetInvalidSession() {
  useAuthStore.getState().clearSession();

  if (typeof window !== "undefined") {
    const path = window.location.pathname;
    if (path.startsWith("/dashboard") || path.startsWith("/admin")) {
      window.location.replace("/");
    }
  }
}

async function refreshAccessToken() {
  const state = useAuthStore.getState();
  if (!state.refreshToken) {
    resetInvalidSession();
    return null;
  }

  const { data } = await authClient.post<AuthResponse>("/auth/refresh", {
    refreshToken: state.refreshToken,
  });

  const nextSession = {
    accessToken: data.accessToken,
    refreshToken: data.refreshToken,
    user: {
      userId: data.userId,
      username: data.username,
      roles: data.roles,
    },
  };

  state.setSession(nextSession);
  return nextSession.accessToken;
}

http.interceptors.response.use(
  (response) => response,
  async (error) => {
    const status = error.response?.status;
    const originalRequest = error.config as RetryableRequestConfig | undefined;
    const requestURL = String(error.config?.url ?? "");

    if (
      status !== 401 ||
      !originalRequest ||
      originalRequest._retry ||
      requestURL.includes("/auth/login") ||
      requestURL.includes("/auth/refresh")
    ) {
      if (status === 401 && requestURL.includes("/auth/refresh")) {
        resetInvalidSession();
      }
      return Promise.reject(error);
    }

    try {
      originalRequest._retry = true;
      refreshPromise ??= refreshAccessToken().finally(() => {
        refreshPromise = null;
      });

      const nextAccessToken = await refreshPromise;
      if (!nextAccessToken) {
        return Promise.reject(error);
      }

      error.config.headers = error.config.headers ?? {};
      error.config.headers.Authorization = `Bearer ${nextAccessToken}`;
      return http(error.config);
    } catch (refreshError) {
      resetInvalidSession();
      return Promise.reject(refreshError);
    }
  },
);

export function getAPIErrorMessage(error: unknown, fallback: string) {
  if (axios.isAxiosError(error)) {
    const responseMessage = error.response?.data?.message;
    if (typeof responseMessage === "string" && responseMessage.trim()) {
      return responseMessage;
    }
  }

  if (error instanceof Error && error.message.trim()) {
    return error.message;
  }

  return fallback;
}

export type MailDeliveryDiagnosticPayload = {
  stage?: string;
  code?: string;
  hint?: string;
  retryable?: boolean;
  message?: string;
};

export function getMailDeliveryDiagnostic(error: unknown): MailDeliveryDiagnosticPayload | null {
  if (!axios.isAxiosError(error)) {
    return null;
  }

  const responseData = error.response?.data;
  if (!responseData || typeof responseData !== "object") {
    return null;
  }

  const payload: MailDeliveryDiagnosticPayload = {};
  if (typeof responseData.stage === "string" && responseData.stage.trim()) {
    payload.stage = responseData.stage.trim();
  }
  if (typeof responseData.code === "string" && responseData.code.trim()) {
    payload.code = responseData.code.trim().toLowerCase();
  }
  if (typeof responseData.hint === "string" && responseData.hint.trim()) {
    payload.hint = responseData.hint.trim();
  }
  if (typeof responseData.retryable === "boolean") {
    payload.retryable = responseData.retryable;
  }
  if (typeof responseData.message === "string" && responseData.message.trim()) {
    payload.message = responseData.message.trim();
  }

  return Object.keys(payload).length ? payload : null;
}

export function getMailDeliveryErrorMessage(error: unknown, fallback: string) {
  const message = getAPIErrorMessage(error, fallback);
  const diagnostic = getMailDeliveryDiagnostic(error);
  const responseCode = diagnostic?.code ?? "";
  const responseHint = diagnostic?.hint ?? "";

  if (responseCode || responseHint) {
    const suffix = responseHint ? ` ${responseHint}` : "";
    switch (responseCode) {
      case "connect_failed":
        return `${message}${suffix || " 请检查 SMTP 主机、端口与网络连通性。"}`;
      case "starttls_unavailable":
      case "tls_failed":
      case "tls_certificate_invalid":
        return `${message}${suffix || " 请检查传输模式、证书配置，或确认服务端是否支持 STARTTLS / SMTPS。"}`;
      case "auth_failed":
      case "auth_unavailable":
        return `${message}${suffix || " 请检查 SMTP 账号密码，或确认服务端已开启 AUTH。"}`;
      case "sender_rejected":
        return `${message}${suffix || " 请确认发件邮箱地址已被该 SMTP 服务商允许作为发信身份。"}`;
      case "recipient_rejected":
        return `${message}${suffix || " 请检查测试收件人地址，或确认服务商没有拒收该目标地址。"}`;
      case "data_failed":
        return `${message}${suffix || " 请检查邮件内容大小限制，或稍后重试。"}`;
      case "quit_failed":
        return `${message}${suffix || " 邮件主体可能已发送，建议先检查收件箱后再重试。"}`;
      case "timeout":
        return `${message}${suffix || " 请检查网络、防火墙或 SMTP 服务端响应速度。"}`;
      default:
        return suffix ? `${message} ${responseHint}` : message;
    }
  }

  const normalized = message.toLowerCase();

  if (normalized.includes("mail delivery connect failed")) {
    return `${message} 请检查 SMTP 主机、端口与网络连通性。`;
  }
  if (
    normalized.includes("mail delivery tls handshake failed") ||
    normalized.includes("starttls") ||
    normalized.includes("certificate")
  ) {
    return `${message} 请检查传输模式、证书配置，或确认服务端是否支持 STARTTLS / SMTPS。`;
  }
  if (
    normalized.includes("mail delivery authentication failed") ||
    normalized.includes("advertise auth")
  ) {
    return `${message} 请检查 SMTP 账号密码，或确认服务端已开启 AUTH。`;
  }
  if (normalized.includes("mail delivery mail from failed")) {
    return `${message} 请确认发件邮箱地址已被该 SMTP 服务商允许作为发信身份。`;
  }
  if (normalized.includes("mail delivery rcpt to failed")) {
    return `${message} 请检查测试收件人地址，或确认服务商没有拒收该目标地址。`;
  }
  if (normalized.includes("mail delivery data failed")) {
    return `${message} 请检查邮件内容大小限制，或稍后重试。`;
  }
  if (normalized.includes("mail delivery quit failed")) {
    return `${message} 邮件主体可能已发送，建议先检查收件箱后再重试。`;
  }
  if (normalized.includes("operation timed out")) {
    return `${message} 请检查网络、防火墙或 SMTP 服务端响应速度。`;
  }

  return message;
}

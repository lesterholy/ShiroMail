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

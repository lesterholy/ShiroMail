export type AuthUser = {
  userId: number;
  username: string;
  roles: string[];
};

export type AuthSession = {
  accessToken: string;
  refreshToken: string;
  user: AuthUser;
};

export type AuthResponse = {
  userId: number;
  username: string;
  roles: string[];
  accessToken: string;
  refreshToken: string;
};

export const AUTH_STORAGE_KEY = "shiro-email.session";

export function getDefaultRouteForRoles(roles: string[]) {
  return roles.includes("admin") ? "/admin" : "/dashboard";
}

export function readStoredSession(): AuthSession | null {
  const raw = window.localStorage.getItem(AUTH_STORAGE_KEY);
  if (!raw) {
    return null;
  }

  try {
    return JSON.parse(raw) as AuthSession;
  } catch {
    window.localStorage.removeItem(AUTH_STORAGE_KEY);
    return null;
  }
}

export function writeStoredSession(session: AuthSession | null) {
  if (!session) {
    window.localStorage.removeItem(AUTH_STORAGE_KEY);
    return;
  }

  window.localStorage.setItem(AUTH_STORAGE_KEY, JSON.stringify(session));
}

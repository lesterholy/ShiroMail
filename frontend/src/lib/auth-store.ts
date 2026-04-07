import { create } from "zustand";
import { type AuthSession, type AuthUser, readStoredSession, writeStoredSession } from "./auth";

type AuthState = {
  accessToken: string | null;
  refreshToken: string | null;
  user: AuthUser | null;
  setSession: (session: AuthSession) => void;
  clearSession: () => void;
};

const initialSession = typeof window === "undefined" ? null : readStoredSession();

export const useAuthStore = create<AuthState>((set) => ({
  accessToken: initialSession?.accessToken ?? null,
  refreshToken: initialSession?.refreshToken ?? null,
  user: initialSession?.user ?? null,
  setSession: (session) => {
    writeStoredSession(session);
    set({
      accessToken: session.accessToken,
      refreshToken: session.refreshToken,
      user: session.user,
    });
  },
  clearSession: () => {
    writeStoredSession(null);
    set({
      accessToken: null,
      refreshToken: null,
      user: null,
    });
  },
}));

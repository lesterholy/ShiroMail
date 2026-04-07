import { getAPIErrorMessage, http } from "@/lib/http";

export type AccountProfile = {
  userId: number;
  username: string;
  email: string;
  emailVerified: boolean;
  roles: string[];
  displayName: string;
  locale: string;
  timezone: string;
  autoRefreshSeconds: number;
  twoFactorEnabled: boolean;
};

export type UpdateAccountProfileInput = {
  displayName: string;
  locale: string;
  timezone: string;
  autoRefreshSeconds: number;
};

export type VerificationChallenge = {
  status: "verification_required";
  email: string;
  verificationTicket: string;
  expiresInSeconds: number;
};

export type ChangePasswordInput = {
  currentPassword: string;
  newPassword: string;
};

export type TOTPStatus = {
  enabled: boolean;
};

export type TOTPSetup = {
  manualEntryKey: string;
  otpauthUrl: string;
};

export async function fetchAccountProfile() {
  const { data } = await http.get<AccountProfile>("/account/profile");
  return data;
}

export async function updateAccountProfile(input: UpdateAccountProfileInput) {
  const { data } = await http.patch<AccountProfile>("/account/profile", input);
  return data;
}

export async function requestAccountEmailChange(newEmail: string) {
  const { data } = await http.post<VerificationChallenge>("/account/email/change/request", {
    newEmail,
  });
  return data;
}

export async function confirmAccountEmailChange(input: {
  verificationTicket: string;
  code: string;
}) {
  const { data } = await http.post<AccountProfile>("/account/email/change/confirm", input);
  return data;
}

export async function changeAccountPassword(input: ChangePasswordInput) {
  const { data } = await http.post<{ status: string }>("/account/password/change", input);
  return data;
}

export async function fetchTOTPStatus() {
  const { data } = await http.get<TOTPStatus>("/account/2fa/status");
  return data;
}

export async function setupTOTP() {
  const { data } = await http.post<TOTPSetup>("/account/2fa/totp/setup", {});
  return data;
}

export async function enableTOTP(code: string) {
  const { data } = await http.post<{ status: string }>("/account/2fa/totp/enable", { code });
  return data;
}

export async function disableTOTP(password: string) {
  const { data } = await http.post<{ status: string }>("/account/2fa/totp/disable", { password });
  return data;
}

export function getAccountErrorMessage(error: unknown, fallback: string) {
  return getAPIErrorMessage(error, fallback);
}

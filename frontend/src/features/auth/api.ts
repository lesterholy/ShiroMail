import { http } from "../../lib/http";
import type { AuthResponse, AuthSession } from "../../lib/auth";
import { getAPIErrorMessage } from "../../lib/http";

export type LoginInput = {
  login: string;
  password: string;
};

export type RegisterInput = {
  username: string;
  email: string;
  password: string;
};

export type ForgotPasswordInput = {
  login: string;
};

export type ForgotPasswordResponse = {
  status: string;
  email: string;
  verificationTicket: string;
  expiresInSeconds: number;
};

export type ResetPasswordInput = {
  verificationTicket: string;
  code: string;
  newPassword: string;
};

export type AuthSettings = {
  registrationMode: string;
  allowRegistration: boolean;
  bootstrapAdminRequired?: boolean;
  requireEmailVerification: boolean;
  inviteOnly: boolean;
  passwordMinLength: number;
  allowMultiSession: boolean;
  refreshTokenDays: number;
  oauthShowOnLogin: boolean;
  oauthProviders: Record<
    string,
    {
      enabled: boolean;
      displayName: string;
      redirectUrl: string;
      authorizationUrl: string;
      scopes: string[];
      usePkce: boolean;
      allowAutoRegister: boolean;
    }
  >;
};

export type OAuthStartResponse = {
  provider: string;
  authorizationUrl: string;
};

export type VerificationChallenge = {
  status: "verification_required";
  email: string;
  verificationTicket: string;
  expiresInSeconds: number;
};

export type TwoFactorChallenge = {
  status: "two_factor_required";
  challengeTicket: string;
  expiresInSeconds: number;
};

export type AuthFlowResult =
  | { kind: "session"; session: AuthSession }
  | { kind: "verification_required"; challenge: VerificationChallenge }
  | { kind: "two_factor_required"; challenge: TwoFactorChallenge };

function isVerificationChallenge(
  value: AuthResponse | VerificationChallenge,
): value is VerificationChallenge {
  return "verificationTicket" in value;
}

function isTwoFactorChallenge(value: unknown): value is TwoFactorChallenge {
  return (
    typeof value === "object" &&
    value !== null &&
    "status" in value &&
    (value as { status?: string }).status === "two_factor_required"
  );
}

function buildSession(data: AuthResponse) {
  return {
    accessToken: data.accessToken,
    refreshToken: data.refreshToken,
    user: {
      userId: data.userId,
      username: data.username,
      roles: data.roles,
    },
  } satisfies AuthSession;
}

export async function login(input: LoginInput) {
  try {
    const { data } = await http.post<AuthResponse>("/auth/login", input);
    return {
      kind: "session",
      session: buildSession(data),
    } satisfies AuthFlowResult;
  } catch (error) {
    const challenge = (error as {
      response?: { data?: VerificationChallenge | TwoFactorChallenge };
    })
      .response?.data;
    if (challenge?.status === "verification_required") {
      return {
        kind: "verification_required",
        challenge,
      } satisfies AuthFlowResult;
    }
    if (isTwoFactorChallenge(challenge)) {
      return {
        kind: "two_factor_required",
        challenge,
      } satisfies AuthFlowResult;
    }
    throw error;
  }
}

export async function register(input: RegisterInput) {
  const { data } = await http.post<AuthResponse | VerificationChallenge>(
    "/auth/register",
    input,
  );
  if (isVerificationChallenge(data)) {
    return {
      kind: "verification_required",
      challenge: data,
    } satisfies AuthFlowResult;
  }
  return {
    kind: "session",
    session: buildSession(data),
  } satisfies AuthFlowResult;
}

export async function fetchAuthSettings() {
  const { data } = await http.get<AuthSettings>("/auth/settings");
  return data;
}

export async function startOAuthLogin(provider: string) {
  const { data } = await http.post<OAuthStartResponse>(
    `/auth/oauth/${provider}/start`,
    {},
  );
  return data;
}

export async function completeOAuthLogin(
  provider: string,
  input: { code: string; state: string },
) {
  const { data } = await http.post<AuthResponse | VerificationChallenge>(
    `/auth/oauth/${provider}/callback`,
    input,
  );
  if (isVerificationChallenge(data)) {
    return {
      kind: "verification_required",
      challenge: data,
    } satisfies AuthFlowResult;
  }
  return {
    kind: "session",
    session: buildSession(data),
  } satisfies AuthFlowResult;
}

export async function requestPasswordReset(input: ForgotPasswordInput) {
  const { data } = await http.post<ForgotPasswordResponse>(
    "/auth/forgot-password",
    input,
  );
  return data;
}

export async function resetPassword(input: ResetPasswordInput) {
  const { data } = await http.post<{ status: string }>(
    "/auth/reset-password",
    input,
  );
  return data;
}

export async function confirmEmailVerification(input: {
  verificationTicket: string;
  code: string;
}) {
  const { data } = await http.post<AuthResponse>(
    "/auth/email-verification/confirm",
    input,
  );
  return buildSession(data);
}

export async function verifyLoginTOTP(input: {
  challengeTicket: string;
  code: string;
}) {
  const { data } = await http.post<AuthResponse>("/auth/login/2fa/verify", input);
  return buildSession(data);
}

export async function resendEmailVerification(input: {
  verificationTicket: string;
}) {
  const { data } = await http.post<VerificationChallenge>(
    "/auth/email-verification/resend",
    input,
  );
  return data;
}

export function getAuthErrorMessage(error: unknown, fallback: string) {
  return getAPIErrorMessage(error, fallback);
}

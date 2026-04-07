export function formatDNSRecordValueForDisplay(
  type: string,
  value: string,
  provider?: string | null,
) {
  const normalizedType = type.trim().toUpperCase();
  const normalizedProvider = provider?.trim().toLowerCase();
  if (normalizedType !== "TXT" || normalizedProvider !== "cloudflare") {
    return value;
  }

  const trimmed = value.trim();
  if (trimmed.startsWith("\"") && trimmed.endsWith("\"") && trimmed.length >= 2) {
    return trimmed;
  }

  return `"${trimmed}"`;
}

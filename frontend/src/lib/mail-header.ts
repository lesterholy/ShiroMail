function decodeBase64Bytes(value: string) {
  const normalized = value.replace(/\s+/g, "");
  const binary = atob(normalized);
  return Uint8Array.from(binary, (char) => char.charCodeAt(0));
}

function decodeQuotedPrintableBytes(value: string) {
  const normalized = value.replace(/_/g, " ");
  const bytes: number[] = [];

  for (let index = 0; index < normalized.length; index += 1) {
    const current = normalized[index];
    if (current === "=" && index + 2 < normalized.length) {
      const hex = normalized.slice(index + 1, index + 3);
      if (/^[0-9a-fA-F]{2}$/.test(hex)) {
        bytes.push(Number.parseInt(hex, 16));
        index += 2;
        continue;
      }
    }
    bytes.push(current.charCodeAt(0));
  }

  return new Uint8Array(bytes);
}

function decodeBytes(charset: string, bytes: Uint8Array) {
  try {
    return new TextDecoder(charset).decode(bytes);
  } catch {
    try {
      return new TextDecoder("utf-8").decode(bytes);
    } catch {
      return String.fromCharCode(...bytes);
    }
  }
}

export function decodeMimeHeaderValue(value: string) {
  if (!value.includes("=?")) {
    return value;
  }

  const collapsed = value.replace(/\?=\s+(=\?)/g, "?=$1");
  return collapsed.replace(/=\?([^?]+)\?([bqBQ])\?([^?]*)\?=/g, (match, charset, encoding, payload) => {
    try {
      const bytes =
        encoding.toUpperCase() === "B"
          ? decodeBase64Bytes(payload)
          : decodeQuotedPrintableBytes(payload);
      return decodeBytes(String(charset).toLowerCase(), bytes);
    } catch {
      return match;
    }
  });
}

import { useEffect } from "react";

const DEFAULT_AUTO_DISMISS_MS = 5000;

export function useAutoDismiss<T>(
  value: T | null | undefined,
  onDismiss: () => void,
  delayMs = DEFAULT_AUTO_DISMISS_MS,
) {
  useEffect(() => {
    if (value == null) {
      return;
    }

    const timer = window.setTimeout(() => {
      onDismiss();
    }, delayMs);

    return () => {
      window.clearTimeout(timer);
    };
  }, [delayMs, onDismiss, value]);
}

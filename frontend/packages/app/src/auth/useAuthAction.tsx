import { useState } from "react";

export const callbackUrl = (path = "/auth/callback") =>
  `${window.location.origin}${path}`;

type Result = Promise<{ error: { message: string } | null }>;

/** Busy/error/message state shared by the sign-in, sign-up and reset pages. */
export function useAuthAction() {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);

  async function run(action: () => Result, onOk?: () => void) {
    setBusy(true);
    setError(null);
    setMessage(null);
    const { error } = await action();
    setBusy(false);
    if (error) setError(error.message);
    else onOk?.();
  }

  const status = (
    <>
      {error && (
        <p role="alert" className="text-sm text-danger">
          {error}
        </p>
      )}
      {message && (
        <p aria-live="polite" className="text-sm text-fg-secondary">
          {message}
        </p>
      )}
    </>
  );

  return { busy, run, setMessage, status };
}

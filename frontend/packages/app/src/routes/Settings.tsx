import { useState } from "react";
import { Button, Card, Tag } from "@bragdoc/ui";
import { errorText } from "../lib/errors";
import {
  useTelegramCode,
  useTelegramStatus,
  useTelegramUnlink,
} from "../settings/useTelegram";

export function Component() {
  return (
    <div className="flex flex-col gap-6">
      <h2 className="type-title-2">Settings</h2>
      <TelegramCard />
    </div>
  );
}

function Alert({ error }: { error: unknown }) {
  if (!error) return null;
  return (
    <p role="alert" className="text-danger">
      {errorText(error)}
    </p>
  );
}

function TelegramCard() {
  const status = useTelegramStatus();
  const code = useTelegramCode();
  const unlink = useTelegramUnlink();
  const [confirming, setConfirming] = useState(false);
  const [copied, setCopied] = useState(false);
  const copy = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
    } catch {
      // ponytail: no clipboard (insecure context, denied); the code stays visible to copy by hand.
    }
  };

  return (
    <Card
      title={
        <span className="flex items-center gap-2">
          Telegram
          {status.data && (
            <Tag tone={status.data.linked ? "success" : "neutral"}>
              {status.data.linked ? "Linked" : "Not linked"}
            </Tag>
          )}
        </span>
      }
    >
      <div className="flex flex-col gap-3 text-sm">
        {status.isPending && <p className="text-fg-secondary">Loading…</p>}
        <Alert error={status.error} />
        {status.data?.linked && (
          <>
            <p>
              {status.data.linked_at
                ? `Linked since ${new Date(status.data.linked_at).toLocaleDateString()}.`
                : "Linked."}{" "}
              Send the bot a message to log it; /help shows the format.
            </p>
            {confirming ? (
              <div className="flex gap-2">
                <Button
                  variant="danger"
                  disabled={unlink.isPending}
                  onClick={() =>
                    unlink.mutate(undefined, {
                      onSuccess: () => {
                        setConfirming(false);
                        code.reset(); // the used code must not reappear
                      },
                    })
                  }
                >
                  Confirm unlink
                </Button>
                <Button variant="tinted" onClick={() => setConfirming(false)}>
                  Cancel
                </Button>
              </div>
            ) : (
              <Button
                variant="glass"
                className="self-start"
                onClick={() => setConfirming(true)}
              >
                Unlink
              </Button>
            )}
            <Alert error={unlink.error} />
          </>
        )}
        {status.data && !status.data.linked && (
          <>
            <p>Link Telegram to add logs by sending the bot a message.</p>
            <Button
              variant="primary"
              className="self-start"
              disabled={code.isPending}
              onClick={() => {
                setCopied(false);
                code.mutate();
              }}
            >
              Generate link code
            </Button>
            {code.data && (
              <div className="flex flex-col gap-2">
                <div className="flex items-center gap-2">
                  <code className="rounded bg-container px-2 py-1 font-mono text-base">
                    {code.data.code}
                  </code>
                  <Button
                    variant="tinted"
                    size="sm"
                    aria-label="Copy link code"
                    onClick={() => void copy(code.data.code)}
                  >
                    {copied ? "Copied" : "Copy"}
                  </Button>
                </div>
                <p className="text-fg-secondary">
                  Send <code>/start {code.data.code}</code> to the bot. Expires
                  at {new Date(code.data.expires_at).toLocaleTimeString()}.
                </p>
                {code.data.bot_url && (
                  <a
                    href={code.data.bot_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="self-start underline"
                  >
                    Open in Telegram
                  </a>
                )}
              </div>
            )}
            <Alert error={code.error} />
          </>
        )}
      </div>
    </Card>
  );
}

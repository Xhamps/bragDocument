import { useState } from "react";
import {
  Badge,
  Button,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@bragdoc/ui";
import { errorText } from "../lib/errors";
import {
  useTelegramCode,
  useTelegramStatus,
  useTelegramUnlink,
} from "../settings/useTelegram";

export function Component() {
  return (
    <div className="flex flex-col gap-6">
      <h2 className="text-xl font-semibold">Settings</h2>
      <TelegramCard />
    </div>
  );
}

function Alert({ error }: { error: unknown }) {
  if (!error) return null;
  return (
    <p role="alert" className="text-destructive">
      {errorText(error)}
    </p>
  );
}

function TelegramCard() {
  const status = useTelegramStatus();
  const code = useTelegramCode();
  const unlink = useTelegramUnlink();
  const [confirming, setConfirming] = useState(false);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          Telegram
          {status.data && (
            <Badge variant={status.data.linked ? "default" : "secondary"}>
              {status.data.linked ? "Linked" : "Not linked"}
            </Badge>
          )}
        </CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3 text-sm">
        {status.isPending && <p className="text-muted-foreground">Loading…</p>}
        <Alert error={status.error} />
        {status.data?.linked && (
          <>
            <p>
              Linked since{" "}
              {status.data.linked_at &&
                new Date(status.data.linked_at).toLocaleDateString()}
              . Send the bot a message to log it; /help shows the format.
            </p>
            {confirming ? (
              <div className="flex gap-2">
                <Button
                  variant="destructive"
                  disabled={unlink.isPending}
                  onClick={() =>
                    unlink.mutate(undefined, {
                      onSuccess: () => setConfirming(false),
                    })
                  }
                >
                  Confirm unlink
                </Button>
                <Button variant="ghost" onClick={() => setConfirming(false)}>
                  Cancel
                </Button>
              </div>
            ) : (
              <Button
                variant="outline"
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
              className="self-start"
              disabled={code.isPending}
              onClick={() => code.mutate()}
            >
              Generate link code
            </Button>
            {code.data && (
              <div className="flex flex-col gap-2">
                <div className="flex items-center gap-2">
                  <code className="rounded bg-muted px-2 py-1 font-mono text-base">
                    {code.data.code}
                  </code>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() =>
                      void navigator.clipboard.writeText(code.data.code)
                    }
                  >
                    Copy
                  </Button>
                </div>
                <p className="text-muted-foreground">
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
      </CardContent>
    </Card>
  );
}

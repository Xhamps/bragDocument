const fmt = new Intl.RelativeTimeFormat(undefined, { numeric: "auto" });

function relative(at: string) {
  // A future time (clock skew) reads as "now".
  const s = Math.max(0, (Date.now() - new Date(at).getTime()) / 1000);
  // Pick the unit on the rounded value: 3570 s is "1 hour ago", not "60 minutes ago".
  const m = Math.round(s / 60);
  if (m < 60) return fmt.format(-m, "minute");
  const h = Math.round(s / 3600);
  if (h < 24) return fmt.format(-h, "hour");
  return fmt.format(-Math.round(s / 86400), "day");
}

/** Local relative time; the exact local timestamp on hover (NFR-4). */
export function When({ at }: { at: string }) {
  return (
    <time dateTime={at} title={new Date(at).toLocaleString()}>
      {relative(at)}
    </time>
  );
}

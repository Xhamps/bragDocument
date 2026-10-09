const fmt = new Intl.RelativeTimeFormat(undefined, { numeric: "auto" });

function relative(at: string) {
  const s = (Date.now() - new Date(at).getTime()) / 1000;
  if (s < 3600) return fmt.format(-Math.round(s / 60), "minute");
  if (s < 86400) return fmt.format(-Math.round(s / 3600), "hour");
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

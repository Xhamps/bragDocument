export type PageItem = number | "gap-l" | "gap-r";

/** Page numbers to show: first, last, `sib` neighbours of `page`, and gaps (the DS pageList, verbatim). */
export function pageList(page: number, count: number, sib: number): PageItem[] {
  const out: PageItem[] = [];
  let lo = Math.max(2, page - sib);
  let hi = Math.min(count - 1, page + sib);
  if (page - sib <= 3) {
    lo = 2;
    hi = Math.min(count - 1, Math.max(hi, 2 + sib * 2));
  }
  if (page + sib >= count - 2) {
    hi = count - 1;
    lo = Math.max(2, Math.min(lo, count - 1 - sib * 2));
  }
  out.push(1);
  if (lo > 2) out.push("gap-l");
  for (let i = lo; i <= hi; i++) out.push(i);
  if (hi < count - 1) out.push("gap-r");
  if (count > 1) out.push(count);
  return out;
}

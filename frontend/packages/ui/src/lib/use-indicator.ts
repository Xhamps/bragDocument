import * as React from "react";

type Pos = { x: number; y: number; w: number; h: number };

// Port of the DS useIndicator: measures the `[data-on="true"]` child of the
// returned ref and positions a sliding pill over it. `animated` turns on after
// the first frame so the initial placement does not slide in from 0,0.
function useIndicator<T extends HTMLElement = HTMLDivElement>(
  deps: React.DependencyList,
) {
  const ref = React.useRef<T>(null);
  const [pos, setPos] = React.useState<Pos | null>(null);
  const [animated, setAnimated] = React.useState(false);

  const measure = React.useCallback(() => {
    const box = ref.current;
    const el = box?.querySelector('[data-on="true"]');
    if (!box || !el) return setPos(null);
    const b = box.getBoundingClientRect();
    const r = el.getBoundingClientRect();
    const next = {
      x: r.left - b.left - box.clientLeft,
      y: r.top - b.top - box.clientTop,
      w: r.width,
      h: r.height,
    };
    // Skip the re-render when resize/font events leave the pill in place.
    setPos((p) =>
      p && p.x === next.x && p.y === next.y && p.w === next.w && p.h === next.h
        ? p
        : next,
    );
  }, []);

  // Callers pass their own deps (value, option count, size); the linter cannot
  // see through a forwarded array.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  React.useLayoutEffect(measure, deps);

  React.useEffect(() => {
    const raf = requestAnimationFrame(() => setAnimated(true));
    window.addEventListener("resize", measure);
    const ro = new ResizeObserver(measure);
    if (ref.current) ro.observe(ref.current);
    void document.fonts?.ready.then(measure);
    return () => {
      cancelAnimationFrame(raf);
      window.removeEventListener("resize", measure);
      ro.disconnect();
    };
  }, [measure]);

  const style: React.CSSProperties = pos
    ? {
        width: pos.w,
        height: pos.h,
        transform: `translate(${pos.x}px, ${pos.y}px)`,
        opacity: 1,
      }
    : { opacity: 0 };
  return { ref, style, animated };
}

export { useIndicator };

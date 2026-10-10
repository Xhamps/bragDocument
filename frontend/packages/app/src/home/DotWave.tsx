import { useEffect, useRef } from "react";

const COLS = 120;
const ROWS = 40;
const DEPTH = 6; // world units from the nearest row to the horizon
const NEAR = 1.2; // camera distance to the nearest row
const SPREAD = 9; // world width of the field
const AMP = 0.35; // wave height
const CAM_Y = 1.1; // camera height above the field
// Pre-baked per-row projection; positions only change with t.
const xs = Float32Array.from({ length: COLS }, (_, i) => i / (COLS - 1) - 0.5);
const zs = Float32Array.from({ length: ROWS }, (_, j) => j / (ROWS - 1));

// Field of glowing dots rippling toward a horizon; decorative only.
export function DotWave({ className }: { className?: string }) {
  const ref = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const canvas = ref.current;
    const ctx = canvas?.getContext("2d");
    const parent = canvas?.parentElement;
    if (!canvas || !ctx || !parent) return; // jsdom has no 2D context

    let w = 0;
    let h = 0;
    let colors: string[] = [];
    let raf = 0;
    let visible = true;
    const reduced = matchMedia("(prefers-reduced-motion: reduce)");

    function readColors() {
      const s = getComputedStyle(document.documentElement);
      colors = ["--chart-3", "--chart-2", "--chart-1"].map(
        (v) => s.getPropertyValue(v).trim() || "#5093f7",
      );
    }

    function draw(t: number) {
      ctx!.clearRect(0, 0, w, h);
      const focal = h * 0.55;
      const horizon = h * 0.42;
      // Far rows first so near (bigger, brighter) dots paint on top.
      for (let j = ROWS - 1; j >= 0; j--) {
        const z = zs[j];
        const scale = focal / (NEAR + z * DEPTH);
        const size = Math.max(1, scale * 0.012);
        const rowAlpha = 0.15 + 0.85 * (1 - z);
        // Three colour bands by depth: violet far, blue near.
        ctx!.fillStyle = colors[Math.min(2, Math.floor((1 - z) * 3))];
        const cz = Math.cos(z * 9 + t * 0.8);
        for (let i = 0; i < COLS; i++) {
          const x = xs[i];
          const wave = Math.sin(x * 11 + t) * cz;
          const sx = w / 2 + x * SPREAD * scale;
          if (sx < -size || sx > w + size) continue;
          const sy = horizon + (CAM_Y - wave * AMP) * scale;
          ctx!.globalAlpha = rowAlpha * (0.45 + 0.55 * ((wave + 1) / 2));
          ctx!.fillRect(sx - size / 2, sy - size / 2, size, size);
        }
      }
      ctx!.globalAlpha = 1;
    }

    function tick(now: number) {
      draw(now / 1000);
      raf = requestAnimationFrame(tick);
    }

    function update() {
      const run = visible && !document.hidden && !reduced.matches;
      if (run && !raf) raf = requestAnimationFrame(tick);
      if (!run && raf) {
        cancelAnimationFrame(raf);
        raf = 0;
      }
      if (!run) draw(performance.now() / 1000);
    }

    function resize() {
      const dpr = Math.min(2, window.devicePixelRatio || 1);
      w = parent!.clientWidth;
      h = parent!.clientHeight;
      canvas!.width = Math.round(w * dpr);
      canvas!.height = Math.round(h * dpr);
      ctx!.setTransform(dpr, 0, 0, dpr, 0, 0);
      if (!raf) draw(performance.now() / 1000);
    }

    readColors();
    resize();
    update();

    const ro = new ResizeObserver(resize);
    ro.observe(parent);
    const io = new IntersectionObserver(([e]) => {
      visible = e.isIntersecting;
      update();
    });
    io.observe(canvas);
    function recolor() {
      readColors();
      if (!raf) draw(performance.now() / 1000);
    }
    // Explicit theme switches and OS scheme changes (when no data-theme is set).
    const mo = new MutationObserver(recolor);
    const scheme = matchMedia("(prefers-color-scheme: dark)");
    mo.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ["data-theme"],
    });
    document.addEventListener("visibilitychange", update);
    reduced.addEventListener("change", update);
    scheme.addEventListener("change", recolor);

    return () => {
      cancelAnimationFrame(raf);
      ro.disconnect();
      io.disconnect();
      mo.disconnect();
      document.removeEventListener("visibilitychange", update);
      reduced.removeEventListener("change", update);
      scheme.removeEventListener("change", recolor);
    };
  }, []);

  return (
    <canvas
      ref={ref}
      aria-hidden
      className={className}
      style={{
        maskImage:
          "linear-gradient(to bottom, transparent, black 15%, black 70%, transparent)",
      }}
    />
  );
}

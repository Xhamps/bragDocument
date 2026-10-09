import "@testing-library/jest-dom/vitest";

afterEach(() => {
  vi.clearAllMocks();
  vi.unstubAllGlobals();
});

// jsdom has no ResizeObserver; Recharts' ResponsiveContainer needs one.
globalThis.ResizeObserver ??= class {
  observe() {}
  unobserve() {}
  disconnect() {}
};

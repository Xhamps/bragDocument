import "@testing-library/jest-dom/vitest";

// jsdom has no ResizeObserver; Radix popper content needs one.
globalThis.ResizeObserver ??= class {
  observe() {}
  unobserve() {}
  disconnect() {}
} as never;

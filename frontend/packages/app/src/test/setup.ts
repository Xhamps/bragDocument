import "@testing-library/jest-dom/vitest";
import { configure } from "@testing-library/react";

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

// ponytail: the first findBy* in a file waits on a lazy route import (route +
// @bragdoc/ui + recharts); on a loaded CI box that cold transform can exceed
// Testing Library's 1s default. 5s ceiling; split heavy routes if it's hit.
configure({ asyncUtilTimeout: 5000 });

import { cn } from "#lib/utils";

test("DS radius, shadow and type utilities merge like built-ins", () => {
  expect(cn("rounded-sm", "rounded-pill")).toBe("rounded-pill");
  expect(cn("type-callout", "type-body")).toBe("type-body");
  expect(cn("text-sm", "type-body")).toBe("type-body");
  expect(cn("shadow-button", "shadow-glow")).toBe("shadow-glow");
  expect(cn("hover:shadow-md", "hover:shadow-glow-strong")).toBe(
    "hover:shadow-glow-strong",
  );
  expect(cn("inset-shadow-sm", "inset-shadow-ds")).toBe("inset-shadow-ds");
});

test("DS type utilities keep colour and weight classes", () => {
  expect(cn("type-body", "text-fg-primary")).toBe("type-body text-fg-primary");
  expect(cn("type-body", "font-semibold")).toBe("type-body font-semibold");
});

import { createCn } from "cn/config";

// Teach the merger the DS utilities from styles.css; unknown classes are kept
// on both sides of a conflict and CSS order would pick the winner.
export const cn = createCn({
  extend: {
    theme: {
      radius: ["pill"],
      shadow: ["glass", "button", "glow", "glow-strong", "cta"],
      "inset-shadow": ["ds"],
    },
    classGroups: {
      "font-size": [
        {
          type: [
            "display",
            "title-1",
            "title-2",
            "title-3",
            "body",
            "callout",
            "footnote",
            "caption",
            "code",
          ],
        },
      ],
    },
  },
});

export const focusRing =
  "outline-hidden focus-visible:outline-2 focus-visible:outline-solid focus-visible:outline-offset-2 focus-visible:outline-focus-ring";

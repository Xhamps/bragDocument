# Slider

A range input with a value bubble above the thumb, as in the price filter ("Up to CA$6000 · Clear").

- **Provide** `min`, `max`, `step`, `value`/`onChange` or `defaultValue`, and `format` for the bubble and screen-reader text (`v => "CA$ " + v`).
- `label` sits top-left; `onClear` adds a "Clear" link top-right; `showValue={false}` hides the bubble.
- The filled part of the track is `button`, the rest `chart-muted`.
- Use it for approximate values; when the exact number matters, use a TextField with `type="number"`.

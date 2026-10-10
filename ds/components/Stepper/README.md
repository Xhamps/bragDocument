# Stepper

A compact counter with a blue plus, the value and a glass minus, as in "Carry-on bag included [+] 1 [−]".

- **Provide** `value`/`onChange` or `defaultValue`, `min`/`max` (defaults 0–99) and a `label` naming what is counted ("Carry-on bags"), used in the buttons' accessible names.
- A button at its limit shows the inactive state.
- For small integer counts only (travellers, bags, seats); type larger numbers into a TextField.

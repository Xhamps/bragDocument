# TopBar

The page header: brand on the left, navigation in the middle with a sliding glass pill under the current link, and search, actions, the user's avatar menu and a call to action on the right, like the Skyscape and DesignCode headers.

- **Provide** `brand` `{name, href, logo}` (no logo: a small accent-gradient mark), `items` (`{label, value, href, icon, children}`), `value`/`onSelect` or `defaultValue`, and any of `search`, `actions` (IconButtons), `user` (UserMenu props) and `cta` (one Button).
- `variant="bar"` (default): a full-width glass bar with a bottom hairline, 72px tall, for apps. `variant="floating"`: a centred glass pill on `shadow-lg`, 60px, for marketing pages.
- `sticky` keeps it on top; once the page scrolls the bar shrinks to 60px and lifts to `shadow-lg` (floating: `shadow-xl`).
- The current link sits on one glass pill that slides between links (`ease-spring`). An item with `children` opens a tooltip-style dropdown under it, pointer arrow included: rows have an icon tile, a label and a one-line description. It opens on hover, click or ArrowDown, closes on Escape, leaving or choosing.
- Under 760px the links and search collapse behind a menu button that opens a sheet listing every link (children indented).
- Keep it to five or six links, one `cta` at most, and the user menu last.

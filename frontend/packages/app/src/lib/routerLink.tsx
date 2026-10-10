import { Link } from "react-router";

/** `renderLink` for @bragdoc/ui components: client-side navigation instead of a full reload. */
export const routerLink = <P extends { href: string }>(
  _item: unknown,
  { href, ...props }: P,
) => <Link to={href} {...props} />;

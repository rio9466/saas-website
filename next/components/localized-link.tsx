import type { AnchorHTMLAttributes, ReactNode } from "react";
import { Link } from "@/i18n/navigation";

/**
 * Renders an in-site link through next-intl's locale-aware `Link` and anything
 * else (`https://`, `mailto:`, `tel:`, protocol-relative `//…`) as a plain
 * anchor. Always use this for backend-provided URLs such as `cta_url` so no
 * bare `/x` link loses its locale prefix (contract §7.3).
 */

const EXTERNAL = /^(?:[a-z][a-z0-9+.-]*:|\/\/)/i;

interface LocalizedLinkProps
  extends Omit<AnchorHTMLAttributes<HTMLAnchorElement>, "href"> {
  href: string;
  children?: ReactNode;
}

export function LocalizedLink({ href, children, ...rest }: LocalizedLinkProps) {
  if (!href || EXTERNAL.test(href)) {
    return (
      <a href={href || "#"} {...rest}>
        {children}
      </a>
    );
  }

  return (
    <Link href={href} {...rest}>
      {children}
    </Link>
  );
}

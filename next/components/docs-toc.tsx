import { useTranslations } from "next-intl";
import type { MarkdownHeading } from "@/lib/markdown";

/** Table of contents for a documentation article (h2/h3 anchors). */
export function DocsToc({ links }: { links: MarkdownHeading[] }) {
  const t = useTranslations("docs");

  return (
    <nav aria-label={t("tableOfContents")}>
      <p className="mb-3 text-sm font-semibold">{t("tableOfContents")}</p>
      <ul className="flex flex-col gap-1 text-sm">
        {links.map((link) => (
          <li key={link.id} className={link.level === 3 ? "pl-3" : ""}>
            <a
              href={`#${link.id}`}
              className="block text-muted-foreground hover:text-foreground"
            >
              {link.text}
            </a>
          </li>
        ))}
      </ul>
    </nav>
  );
}

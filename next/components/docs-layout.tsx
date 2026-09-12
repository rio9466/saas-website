import { useTranslations } from "next-intl";
import { cn } from "cn";
import { Link } from "@/i18n/navigation";
import { DocsToc } from "@/components/docs-toc";
import { MarkdownContent } from "@/components/markdown-content";
import { renderMarkdown } from "@/lib/markdown";
import type { DocArticle, DocCategory } from "@/lib/public-content";

/**
 * Documentation shell: category sidebar (with the active category
 * highlighted), article body, TOC (desktop + mobile) and prev/next links.
 */
export function DocsLayout({
  categories,
  article,
}: {
  categories: DocCategory[];
  article: DocArticle;
}) {
  const t = useTranslations();
  const toc = renderMarkdown(article.body_md).toc.filter(
    (heading) => heading.level >= 2 && heading.level <= 3,
  );
  const activeCategorySlug = article.category?.slug ?? "";

  return (
    <div className="mx-auto grid w-full max-w-7xl grid-cols-1 gap-10 px-4 py-10 lg:grid-cols-[16rem_minmax(0,1fr)] xl:grid-cols-[16rem_minmax(0,1fr)_14rem]">
      <aside className="lg:sticky lg:top-24 lg:self-start">
        <nav className="flex flex-col gap-6" aria-label={t("docs.navigation")}>
          {categories.map((category) => (
            <div key={category.id} className="flex flex-col gap-2">
              <p
                className={cn(
                  "text-xs font-semibold tracking-wide uppercase",
                  category.slug === activeCategorySlug
                    ? "text-primary"
                    : "text-muted-foreground",
                )}
              >
                {category.name}
              </p>
              <ul className="flex flex-col gap-0.5">
                {category.articles.map((item) => (
                  <li key={item.id}>
                    <Link
                      href={`/docs/${item.slug}`}
                      className={cn(
                        "block rounded-md px-3 py-1.5 text-sm transition-colors",
                        item.slug === article.slug
                          ? "bg-muted font-medium text-primary"
                          : "text-muted-foreground hover:bg-muted hover:text-foreground",
                      )}
                    >
                      {item.title}
                    </Link>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </nav>
      </aside>

      <article className="min-w-0">
        <nav
          className="flex flex-wrap items-center gap-1 text-sm text-muted-foreground"
          aria-label={t("docs.breadcrumb")}
        >
          <Link href="/docs" className="hover:text-foreground">
            {t("nav.docs")}
          </Link>
          {article.category?.name ? <span>/</span> : null}
          {article.category?.name ? <span>{article.category.name}</span> : null}
        </nav>

        <h1 className="mt-3 font-heading text-3xl font-bold">{article.title}</h1>

        <MarkdownContent content={article.body_md} className="mt-6" />

        {toc.length ? (
          <div className="mt-10 rounded-xl border bg-muted/30 p-4 xl:hidden">
            <DocsToc links={toc} />
          </div>
        ) : null}

        {article.prev || article.next ? (
          <nav
            className="mt-12 grid gap-4 border-t pt-6 sm:grid-cols-2"
            aria-label={t("docs.surround")}
          >
            {article.prev ? (
              <Link
                href={`/docs/${article.prev.slug}`}
                className="flex flex-col gap-1 rounded-lg border p-4 hover:border-primary"
              >
                <span className="text-xs text-muted-foreground">
                  {t("docs.previous")}
                </span>
                <span className="font-medium">{article.prev.title}</span>
              </Link>
            ) : null}
            {article.next ? (
              <Link
                href={`/docs/${article.next.slug}`}
                className="flex flex-col gap-1 rounded-lg border p-4 hover:border-primary sm:col-start-2 sm:text-right"
              >
                <span className="text-xs text-muted-foreground">
                  {t("docs.next")}
                </span>
                <span className="font-medium">{article.next.title}</span>
              </Link>
            ) : null}
          </nav>
        ) : null}
      </article>

      {toc.length ? (
        <aside className="hidden xl:sticky xl:top-24 xl:block xl:self-start">
          <DocsToc links={toc} />
        </aside>
      ) : null}
    </div>
  );
}

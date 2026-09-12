import type { Metadata } from "next";
import { hasLocale } from "next-intl";
import { getTranslations, setRequestLocale } from "next-intl/server";
import { notFound } from "next/navigation";
import { DocsLayout } from "@/components/docs-layout";
import { firstDocArticle, getDoc, getDocs } from "@/lib/public-content";
import { pageMetadata } from "@/lib/seo";
import { routing } from "@/i18n/routing";

interface LocaleRouteProps {
  params: Promise<{ locale: string }>;
}

export async function generateMetadata({
  params,
}: LocaleRouteProps): Promise<Metadata> {
  const requested = (await params).locale;
  const locale = hasLocale(routing.locales, requested)
    ? requested
    : routing.defaultLocale;
  const t = await getTranslations({ locale });
  return pageMetadata({
    locale,
    path: "/docs",
    title: t("docs.title"),
    description: t("docs.subtitle"),
  });
}

/**
 * Documentation landing page: shows the first article of the tree when one
 * exists, otherwise an empty state.
 */
export default async function DocsPage({ params }: LocaleRouteProps) {
  const { locale } = await params;
  if (!hasLocale(routing.locales, locale)) {
    notFound();
  }
  setRequestLocale(locale);

  const t = await getTranslations({ locale });
  const categories = await getDocs(locale);
  const first = firstDocArticle(categories);
  const article = first ? await getDoc(locale, first.slug) : null;

  if (article) {
    return <DocsLayout categories={categories} article={article} />;
  }

  return (
    <div className="mx-auto w-full max-w-3xl px-4 py-16">
      <h1 className="font-heading text-3xl font-semibold tracking-tight">
        {t("docs.title")}
      </h1>
      <p className="mt-4 rounded-xl border bg-muted/30 px-6 py-10 text-center text-muted-foreground">
        {t("docs.empty")}
      </p>
    </div>
  );
}

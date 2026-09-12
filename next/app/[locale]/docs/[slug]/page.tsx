import type { Metadata } from "next";
import { hasLocale } from "next-intl";
import { setRequestLocale } from "next-intl/server";
import { notFound } from "next/navigation";
import { DocsLayout } from "@/components/docs-layout";
import { getDoc, getDocs } from "@/lib/public-content";
import { pageMetadata } from "@/lib/seo";
import { routing } from "@/i18n/routing";

interface DocRouteProps {
  params: Promise<{ locale: string; slug: string }>;
}

export async function generateMetadata({
  params,
}: DocRouteProps): Promise<Metadata> {
  const { locale: requested, slug } = await params;
  const locale = hasLocale(routing.locales, requested)
    ? requested
    : routing.defaultLocale;
  const article = await getDoc(locale, slug);
  if (!article) {
    return {};
  }
  return pageMetadata({
    locale,
    path: `/docs/${slug}`,
    title: article.seo_title || article.title,
    description: article.seo_description || undefined,
  });
}

/**
 * One documentation article. An unpublished or unknown slug renders the 404
 * page (contract §4.7).
 */
export default async function DocArticlePage({ params }: DocRouteProps) {
  const { locale, slug } = await params;
  if (!hasLocale(routing.locales, locale)) {
    notFound();
  }
  setRequestLocale(locale);

  const [categories, article] = await Promise.all([
    getDocs(locale),
    getDoc(locale, slug),
  ]);

  if (!article) {
    notFound();
  }

  return <DocsLayout categories={categories} article={article} />;
}

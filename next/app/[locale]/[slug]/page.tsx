import type { Metadata } from "next";
import { hasLocale } from "next-intl";
import { setRequestLocale } from "next-intl/server";
import { notFound } from "next/navigation";
import { MarkdownContent } from "@/components/markdown-content";
import { getPage } from "@/lib/public-content";
import { pageMetadata } from "@/lib/seo";
import { routing } from "@/i18n/routing";

interface PageRouteProps {
  params: Promise<{ locale: string; slug: string }>;
}

export async function generateMetadata({
  params,
}: PageRouteProps): Promise<Metadata> {
  const { locale: requested, slug } = await params;
  const locale = hasLocale(routing.locales, requested)
    ? requested
    : routing.defaultLocale;
  const page = await getPage(locale, slug);
  if (!page) {
    return {};
  }
  return pageMetadata({
    locale,
    path: `/${slug}`,
    title: page.seo_title || page.title,
    description: page.seo_description || undefined,
  });
}

/**
 * Any published content page, e.g. `/about` (contract §4.6). Static routes
 * such as `/features` take precedence; an unpublished or unknown slug renders
 * the 404 page.
 */
export default async function ContentPage({ params }: PageRouteProps) {
  const { locale, slug } = await params;
  if (!hasLocale(routing.locales, locale)) {
    notFound();
  }
  setRequestLocale(locale);

  const page = await getPage(locale, slug);
  if (!page) {
    notFound();
  }

  return (
    <div className="mx-auto w-full max-w-3xl px-4 py-16">
      <article>
        <h1 className="font-heading text-3xl font-semibold tracking-tight sm:text-4xl">
          {page.title}
        </h1>
        <MarkdownContent content={page.body_md} className="mt-6" />
      </article>
    </div>
  );
}

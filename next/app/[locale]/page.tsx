import type { Metadata } from "next";
import { hasLocale } from "next-intl";
import { getTranslations, setRequestLocale } from "next-intl/server";
import { notFound } from "next/navigation";
import { HomeSection, KNOWN_SECTION_TYPES } from "@/components/home-section";
import { LocalizedLink } from "@/components/localized-link";
import { buttonVariants } from "@/components/ui/button";
import { getFeatures, getHomeSections } from "@/lib/public-content";
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
  return pageMetadata({ locale, path: "/" });
}

/**
 * Content-driven home page (contract §4.3). Sections come from
 * `GET /public/home`; unknown types are ignored for forward compatibility, and
 * an empty result falls back to a built-in hero.
 */
export default async function HomePage({ params }: LocaleRouteProps) {
  const { locale } = await params;
  if (!hasLocale(routing.locales, locale)) {
    notFound();
  }
  setRequestLocale(locale);

  const t = await getTranslations({ locale });
  const [sections, features] = await Promise.all([
    getHomeSections(locale),
    getFeatures(locale),
  ]);
  const known = sections.filter((section) =>
    KNOWN_SECTION_TYPES.has(section.type),
  );

  if (!known.length) {
    return (
      <section className="mx-auto w-full max-w-4xl px-4 py-24 text-center sm:py-32">
        <h1 className="font-heading text-4xl font-semibold tracking-tight sm:text-5xl">
          {t("home.fallbackTitle")}
        </h1>
        <p className="mx-auto mt-4 max-w-2xl text-lg text-muted-foreground">
          {t("home.fallbackDescription")}
        </p>
        <div className="mt-8 flex flex-wrap items-center justify-center gap-3">
          <LocalizedLink
            href="/register"
            className={buttonVariants({ size: "lg" })}
          >
            {t("home.fallbackPrimary")}
          </LocalizedLink>
          <LocalizedLink
            href="/contact"
            className={buttonVariants({ variant: "outline", size: "lg" })}
          >
            {t("home.fallbackSecondary")}
          </LocalizedLink>
        </div>
      </section>
    );
  }

  return (
    <>
      {known.map((section) => (
        <HomeSection key={section.id} section={section} features={features} />
      ))}
    </>
  );
}

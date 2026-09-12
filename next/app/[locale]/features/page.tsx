import type { Metadata } from "next";
import { hasLocale } from "next-intl";
import { getTranslations, setRequestLocale } from "next-intl/server";
import { notFound } from "next/navigation";
import { FeatureCard } from "@/components/feature-card";
import { getFeatures } from "@/lib/public-content";
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
    path: "/features",
    title: t("features.title"),
    description: t("features.subtitle"),
  });
}

export default async function FeaturesPage({ params }: LocaleRouteProps) {
  const { locale } = await params;
  if (!hasLocale(routing.locales, locale)) {
    notFound();
  }
  setRequestLocale(locale);

  const t = await getTranslations({ locale });
  const features = await getFeatures(locale);

  return (
    <div className="mx-auto w-full max-w-6xl px-4 py-16">
      <h1 className="font-heading text-3xl font-semibold tracking-tight sm:text-4xl">
        {t("features.title")}
      </h1>
      <p className="mt-2 max-w-2xl text-muted-foreground">
        {t("features.subtitle")}
      </p>

      {features.length ? (
        <div className="mt-10 grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
          {features.map((feature) => (
            <FeatureCard key={feature.id} feature={feature} />
          ))}
        </div>
      ) : (
        <p className="mt-10 rounded-xl border bg-muted/30 px-6 py-10 text-center text-muted-foreground">
          {t("features.empty")}
        </p>
      )}
    </div>
  );
}

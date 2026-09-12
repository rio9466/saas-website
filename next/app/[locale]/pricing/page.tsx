import type { Metadata } from "next";
import { hasLocale } from "next-intl";
import { getTranslations, setRequestLocale } from "next-intl/server";
import { notFound } from "next/navigation";
import { PricingPlans } from "@/components/pricing-plans";
import { getPricing } from "@/lib/public-content";
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
    path: "/pricing",
    title: t("pricing.title"),
    description: t("pricing.subtitle"),
  });
}

export default async function PricingPage({ params }: LocaleRouteProps) {
  const { locale } = await params;
  if (!hasLocale(routing.locales, locale)) {
    notFound();
  }
  setRequestLocale(locale);

  const t = await getTranslations({ locale });
  const pricing = await getPricing(locale);

  return (
    <div className="mx-auto w-full max-w-6xl px-4 py-16">
      <div className="text-center">
        <h1 className="font-heading text-3xl font-semibold tracking-tight sm:text-4xl">
          {t("pricing.title")}
        </h1>
        <p className="mx-auto mt-2 max-w-2xl text-muted-foreground">
          {t("pricing.subtitle")}
        </p>
      </div>

      {pricing.plans.length ? (
        <PricingPlans plans={pricing.plans} currency={pricing.currency} />
      ) : (
        <p className="mt-10 rounded-xl border bg-muted/30 px-6 py-10 text-center text-muted-foreground">
          {t("pricing.empty")}
        </p>
      )}
    </div>
  );
}

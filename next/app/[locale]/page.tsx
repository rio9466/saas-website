import { hasLocale } from "next-intl";
import { getTranslations, setRequestLocale } from "next-intl/server";
import { Link } from "@/i18n/navigation";
import { buttonVariants } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { getSiteSettings } from "@/lib/site-settings";
import { routing } from "@/i18n/routing";

interface LocaleRouteProps {
  params: Promise<{ locale: string }>;
}

/**
 * Foundation placeholder home page. The real, content-driven home page is
 * built in NEXT-02; this only proves the shell, i18n and settings pipeline.
 */
export default async function HomePage({ params }: LocaleRouteProps) {
  const { locale } = await params;
  if (!hasLocale(routing.locales, locale)) {
    return null;
  }

  setRequestLocale(locale);

  const t = await getTranslations({ locale });
  const settings = await getSiteSettings(locale, t);

  return (
    <section className="mx-auto w-full max-w-6xl px-4 py-16">
      <div className="mx-auto max-w-2xl text-center">
        <h1 className="font-heading text-4xl font-semibold tracking-tight">
          {settings.site_name}
        </h1>
        {settings.tagline ? (
          <p className="mt-4 text-lg text-muted-foreground">
            {settings.tagline}
          </p>
        ) : null}
        <div className="mt-8 flex flex-wrap justify-center gap-3">
          <Link href="/register" className={buttonVariants({ size: "lg" })}>
            {t("home.primaryCta")}
          </Link>
          <Link
            href="/contact"
            className={buttonVariants({ variant: "outline", size: "lg" })}
          >
            {t("home.secondaryCta")}
          </Link>
        </div>
      </div>

      <Card className="mx-auto mt-12 max-w-xl">
        <CardHeader>
          <CardTitle>{settings.site_name}</CardTitle>
          <CardDescription>{t("home.placeholder")}</CardDescription>
        </CardHeader>
        <CardContent className="text-sm text-muted-foreground">
          {settings.contact_email ? (
            <p>
              <a href={`mailto:${settings.contact_email}`}>
                {settings.contact_email}
              </a>
            </p>
          ) : null}
        </CardContent>
      </Card>
    </section>
  );
}

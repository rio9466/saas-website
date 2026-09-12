import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { hasLocale, NextIntlClientProvider } from "next-intl";
import { getTranslations, setRequestLocale } from "next-intl/server";
import { routing } from "@/i18n/routing";
import { SiteHeader } from "@/components/site-header";
import { SiteFooter } from "@/components/site-footer";
import { PageViewReporter } from "@/components/page-view-reporter";
import { ThemeProvider } from "@/components/theme-provider";
import { getSiteSettings } from "@/lib/site-settings";
import { buildMetadata } from "@/lib/seo";
import "../globals.css";

interface LocaleRouteProps {
  params: Promise<{ locale: string }>;
}

export function generateStaticParams() {
  return routing.locales.map((locale) => ({ locale }));
}

export async function generateMetadata({
  params,
}: LocaleRouteProps): Promise<Metadata> {
  const requested = (await params).locale;
  const locale = hasLocale(routing.locales, requested)
    ? requested
    : routing.defaultLocale;
  const t = await getTranslations({ locale });
  const settings = await getSiteSettings(locale, t);
  return buildMetadata({ locale, settings });
}

export default async function LocaleLayout({
  children,
  params,
}: LocaleRouteProps & { children: React.ReactNode }) {
  const { locale } = await params;
  if (!hasLocale(routing.locales, locale)) {
    notFound();
  }

  // Required for static rendering of the locale segment.
  setRequestLocale(locale);

  const t = await getTranslations({ locale });
  const settings = await getSiteSettings(locale, t);

  return (
    <html lang={locale} className="h-full" suppressHydrationWarning>
      <body className="flex min-h-full flex-col antialiased">
        <NextIntlClientProvider>
          <ThemeProvider
            attribute="class"
            defaultTheme="system"
            enableSystem
            disableTransitionOnChange
          >
            <SiteHeader settings={settings} />
            <main className="flex-1">{children}</main>
            <SiteFooter settings={settings} />
            <PageViewReporter />
          </ThemeProvider>
        </NextIntlClientProvider>
      </body>
    </html>
  );
}

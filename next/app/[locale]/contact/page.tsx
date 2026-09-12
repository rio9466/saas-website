import type { Metadata } from "next";
import { hasLocale } from "next-intl";
import { getTranslations, setRequestLocale } from "next-intl/server";
import { notFound } from "next/navigation";
import { Mail, MapPin, Phone } from "lucide-react";
import { ContactForm } from "@/components/contact-form";
import { getSiteSettings } from "@/lib/site-settings";
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
    path: "/contact",
    title: t("contact.title"),
    description: t("contact.subtitle"),
  });
}

export default async function ContactPage({ params }: LocaleRouteProps) {
  const { locale } = await params;
  if (!hasLocale(routing.locales, locale)) {
    notFound();
  }
  setRequestLocale(locale);

  const t = await getTranslations({ locale });
  const settings = await getSiteSettings(locale, t);
  const hasContactInfo = Boolean(
    settings.contact_email ||
      settings.contact_phone ||
      settings.contact_address ||
      settings.social_links.length,
  );

  return (
    <div className="mx-auto w-full max-w-5xl px-4 py-16">
      <h1 className="font-heading text-3xl font-semibold tracking-tight sm:text-4xl">
        {t("contact.title")}
      </h1>
      <p className="mt-2 max-w-2xl text-muted-foreground">
        {t("contact.subtitle")}
      </p>

      <div className="mt-10 grid gap-10 lg:grid-cols-[minmax(0,1fr)_18rem]">
        <ContactForm />

        {hasContactInfo ? (
          <aside className="flex flex-col gap-4">
            <h2 className="text-sm font-semibold tracking-wide text-muted-foreground uppercase">
              {t("contact.infoTitle")}
            </h2>

            {settings.contact_email ? (
              <a
                href={`mailto:${settings.contact_email}`}
                className="flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground"
              >
                <Mail className="size-4" aria-hidden="true" />
                {settings.contact_email}
              </a>
            ) : null}

            {settings.contact_phone ? (
              <a
                href={`tel:${settings.contact_phone}`}
                className="flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground"
              >
                <Phone className="size-4" aria-hidden="true" />
                {settings.contact_phone}
              </a>
            ) : null}

            {settings.contact_address ? (
              <p className="flex items-start gap-2 text-sm text-muted-foreground">
                <MapPin className="mt-0.5 size-4 shrink-0" aria-hidden="true" />
                {settings.contact_address}
              </p>
            ) : null}

            {settings.social_links.length ? (
              <div className="flex flex-col gap-2 pt-2">
                {settings.social_links.map((link) => (
                  <a
                    key={`${link.platform}-${link.url}`}
                    href={link.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-sm text-muted-foreground hover:text-foreground"
                  >
                    {link.platform}
                  </a>
                ))}
              </div>
            ) : null}
          </aside>
        ) : null}
      </div>
    </div>
  );
}

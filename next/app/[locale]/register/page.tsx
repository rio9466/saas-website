import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { hasLocale } from "next-intl";
import { getTranslations, setRequestLocale } from "next-intl/server";
import { routing } from "@/i18n/routing";
import { getSiteSettings } from "@/lib/site-settings";
import { AuthContainer } from "@/components/auth/auth-container";
import { RegisterForm } from "@/components/auth/register-form";

interface PageProps {
  params: Promise<{ locale: string }>;
}

export async function generateMetadata({
  params,
}: PageProps): Promise<Metadata> {
  const { locale } = await params;
  if (!hasLocale(routing.locales, locale)) {
    return {};
  }
  const t = await getTranslations({ locale });
  return {
    title: t("auth.register.title"),
    description: t("auth.register.subtitle"),
  };
}

export default async function RegisterPage({ params }: PageProps) {
  const { locale } = await params;
  if (!hasLocale(routing.locales, locale)) {
    notFound();
  }
  setRequestLocale(locale);

  const t = await getTranslations({ locale });
  const settings = await getSiteSettings(locale, t);

  return (
    <AuthContainer
      title={t("auth.register.title")}
      description={t("auth.register.subtitle")}
    >
      <RegisterForm registrationEnabled={settings.registration_enabled} />
    </AuthContainer>
  );
}

import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { hasLocale } from "next-intl";
import { getTranslations, setRequestLocale } from "next-intl/server";
import { routing } from "@/i18n/routing";
import { getSiteSettings } from "@/lib/site-settings";
import { AuthContainer } from "@/components/auth/auth-container";
import { LoginForm } from "@/components/auth/login-form";

interface PageProps {
  params: Promise<{ locale: string }>;
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}

function firstString(value: string | string[] | undefined): string {
  if (Array.isArray(value)) {
    return value[0] ?? "";
  }
  return value ?? "";
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
    title: t("auth.login.title"),
    description: t("auth.login.subtitle"),
  };
}

export default async function LoginPage({ params, searchParams }: PageProps) {
  const { locale } = await params;
  if (!hasLocale(routing.locales, locale)) {
    notFound();
  }
  setRequestLocale(locale);

  const t = await getTranslations({ locale });
  const query = await searchParams;
  const settings = await getSiteSettings(locale, t);

  return (
    <AuthContainer
      title={t("auth.login.title")}
      description={t("auth.login.subtitle")}
    >
      <LoginForm
        redirect={firstString(query.redirect) || null}
        initialIdentifier={firstString(query.identifier)}
        registered={firstString(query.registered) === "1"}
        passwordReset={firstString(query.reset) === "1"}
        passwordChanged={firstString(query.changed) === "1"}
        registrationEnabled={settings.registration_enabled}
        usernameLoginEnabled={settings.username_login_enabled}
        emailLoginEnabled={settings.email_login_enabled}
      />
    </AuthContainer>
  );
}

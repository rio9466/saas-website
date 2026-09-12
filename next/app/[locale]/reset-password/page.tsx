import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { hasLocale } from "next-intl";
import { getTranslations, setRequestLocale } from "next-intl/server";
import { routing } from "@/i18n/routing";
import { AuthContainer } from "@/components/auth/auth-container";
import { ResetPasswordForm } from "@/components/auth/reset-password-form";

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
    title: t("auth.reset.title"),
    description: t("auth.reset.subtitle"),
  };
}

export default async function ResetPasswordPage({
  params,
  searchParams,
}: PageProps) {
  const { locale } = await params;
  if (!hasLocale(routing.locales, locale)) {
    notFound();
  }
  setRequestLocale(locale);

  const t = await getTranslations({ locale });
  const query = await searchParams;

  return (
    <AuthContainer
      title={t("auth.reset.title")}
      description={t("auth.reset.subtitle")}
    >
      <ResetPasswordForm
        email={firstString(query.email).trim()}
        token={firstString(query.token).trim()}
      />
    </AuthContainer>
  );
}

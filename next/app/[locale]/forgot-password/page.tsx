import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { hasLocale } from "next-intl";
import { getTranslations, setRequestLocale } from "next-intl/server";
import { routing } from "@/i18n/routing";
import { AuthContainer } from "@/components/auth/auth-container";
import { ForgotPasswordForm } from "@/components/auth/forgot-password-form";

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
    title: t("auth.forgot.title"),
    description: t("auth.forgot.subtitle"),
  };
}

export default async function ForgotPasswordPage({ params }: PageProps) {
  const { locale } = await params;
  if (!hasLocale(routing.locales, locale)) {
    notFound();
  }
  setRequestLocale(locale);

  const t = await getTranslations({ locale });

  return (
    <AuthContainer
      title={t("auth.forgot.title")}
      description={t("auth.forgot.subtitle")}
    >
      <ForgotPasswordForm />
    </AuthContainer>
  );
}

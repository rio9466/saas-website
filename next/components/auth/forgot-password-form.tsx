"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { Link } from "@/i18n/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useAuth } from "@/lib/auth";
import { ApiError, apiErrorKey } from "@/lib/api-error";
import { EMAIL_PATTERN } from "@/lib/format";
import { FormAlert } from "./form-alert";
import { FormField } from "./form-field";

export function ForgotPasswordForm() {
  const t = useTranslations();
  const { forgotPassword } = useAuth();

  const [email, setEmail] = useState("");
  const [fieldError, setFieldError] = useState("");
  const [formError, setFormError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [sent, setSent] = useState(false);

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setFormError("");
    setFieldError("");
    if (!email.trim()) {
      setFieldError(t("auth.forgot.errors.emailRequired"));
      return;
    }
    if (!EMAIL_PATTERN.test(email.trim())) {
      setFieldError(t("auth.forgot.errors.emailInvalid"));
      return;
    }
    setSubmitting(true);
    try {
      await forgotPassword(email.trim());
      setSent(true);
    } catch (error) {
      setFormError(
        error instanceof ApiError && error.code === 10001
          ? t("auth.forgot.errors.emailInvalid")
          : error instanceof ApiError
            ? t(apiErrorKey(error.code))
            : t("errorCodes.unknown"),
      );
    } finally {
      setSubmitting(false);
    }
  }

  if (sent) {
    return (
      <div className="mt-8">
        <FormAlert
          tone="success"
          title={t("auth.forgot.sentTitle")}
          description={t("auth.forgot.sentDescription")}
          action={
            <Link
              href="/login"
              className="text-sm text-primary hover:underline"
            >
              {t("auth.forgot.backToLogin")}
            </Link>
          }
        />
      </div>
    );
  }

  return (
    <form className="mt-8 flex flex-col gap-5" noValidate onSubmit={submit}>
      {formError ? <FormAlert description={formError} /> : null}

      <FormField
        label={t("auth.forgot.email")}
        htmlFor="forgot-email"
        error={fieldError}
      >
        <Input
          id="forgot-email"
          type="email"
          autoComplete="email"
          value={email}
          onChange={(event) => setEmail(event.target.value)}
        />
      </FormField>

      <Button type="submit" size="lg" className="w-full" disabled={submitting}>
        {t("auth.forgot.submit")}
      </Button>

      <p className="text-center text-sm text-muted-foreground">
        <Link href="/login" className="text-primary hover:underline">
          {t("auth.forgot.backToLogin")}
        </Link>
      </p>
    </form>
  );
}

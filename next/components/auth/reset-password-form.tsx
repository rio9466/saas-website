"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { Link, useRouter } from "@/i18n/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useAuth } from "@/lib/auth";
import { ApiError, apiErrorKey } from "@/lib/api-error";
import { byteLength } from "@/lib/format";
import { FormAlert } from "./form-alert";
import { FormField } from "./form-field";

export function ResetPasswordForm({
  email,
  token,
}: {
  email: string;
  token: string;
}) {
  const t = useTranslations();
  const router = useRouter();
  const { resetPassword } = useAuth();

  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [fieldErrors, setFieldErrors] = useState<{
    password?: string;
    confirm?: string;
  }>({});
  const [formError, setFormError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const invalidLink = !email || !token;

  function validate(): boolean {
    const next: typeof fieldErrors = {};
    const passwordBytes = byteLength(password);
    if (!password) {
      next.password = t("auth.reset.errors.passwordRequired");
    } else if (passwordBytes < 8 || passwordBytes > 72) {
      next.password = t("auth.reset.errors.passwordLength");
    }
    if (!confirm) {
      next.confirm = t("auth.reset.errors.confirmRequired");
    } else if (confirm !== password) {
      next.confirm = t("auth.reset.errors.confirmMismatch");
    }
    setFieldErrors(next);
    return Object.keys(next).length === 0;
  }

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setFormError("");
    if (invalidLink || !validate()) {
      return;
    }
    setSubmitting(true);
    try {
      await resetPassword(email, token, password);
      router.push({ pathname: "/login", query: { reset: "1" } });
    } catch (error) {
      setFormError(
        error instanceof ApiError && error.code === 40016
          ? t("auth.reset.invalidDescription")
          : error instanceof ApiError
            ? t(apiErrorKey(error.code))
            : t("errorCodes.unknown"),
      );
    } finally {
      setSubmitting(false);
    }
  }

  if (invalidLink) {
    return (
      <div className="mt-8">
        <FormAlert
          title={t("auth.reset.invalidTitle")}
          description={t("auth.reset.invalidDescription")}
          action={
            <Link
              href="/forgot-password"
              className="text-sm text-primary hover:underline"
            >
              {t("auth.reset.requestNew")}
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
        label={t("auth.reset.fields.password")}
        htmlFor="reset-password"
        hint={t("auth.reset.fields.passwordHint")}
        error={fieldErrors.password}
      >
        <Input
          id="reset-password"
          type="password"
          autoComplete="new-password"
          value={password}
          onChange={(event) => setPassword(event.target.value)}
        />
      </FormField>

      <FormField
        label={t("auth.reset.fields.confirmPassword")}
        htmlFor="reset-confirm"
        error={fieldErrors.confirm}
      >
        <Input
          id="reset-confirm"
          type="password"
          autoComplete="new-password"
          value={confirm}
          onChange={(event) => setConfirm(event.target.value)}
        />
      </FormField>

      <Button type="submit" size="lg" className="w-full" disabled={submitting}>
        {t("auth.reset.submit")}
      </Button>
    </form>
  );
}

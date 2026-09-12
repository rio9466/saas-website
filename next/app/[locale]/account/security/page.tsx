"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { useRouter } from "@/i18n/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useAuth } from "@/lib/auth";
import { ApiError, apiErrorKey } from "@/lib/api-error";
import { byteLength } from "@/lib/format";
import { FormAlert } from "@/components/auth/form-alert";
import { FormField } from "@/components/auth/form-field";

export default function AccountSecurityPage() {
  const t = useTranslations();
  const router = useRouter();
  const { changePassword } = useAuth();

  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [confirm, setConfirm] = useState("");
  const [fieldErrors, setFieldErrors] = useState<{
    current?: string;
    next?: string;
    confirm?: string;
  }>({});
  const [formError, setFormError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  function validate(): boolean {
    const errors: typeof fieldErrors = {};
    if (!current) {
      errors.current = t("account.security.errors.currentRequired");
    }
    const nextBytes = byteLength(next);
    if (!next) {
      errors.next = t("account.security.errors.newRequired");
    } else if (nextBytes < 8 || nextBytes > 72) {
      errors.next = t("account.security.errors.newLength");
    }
    if (!confirm) {
      errors.confirm = t("account.security.errors.confirmRequired");
    } else if (confirm !== next) {
      errors.confirm = t("account.security.errors.confirmMismatch");
    }
    setFieldErrors(errors);
    return Object.keys(errors).length === 0;
  }

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setFormError("");
    if (!validate()) {
      return;
    }
    setSubmitting(true);
    try {
      await changePassword(current, next);
      router.push({ pathname: "/login", query: { changed: "1" } });
    } catch (error) {
      if (error instanceof ApiError && error.code === 40005) {
        setFieldErrors((previous) => ({
          ...previous,
          current: t("errorCodes.40005"),
        }));
      } else {
        setFormError(
          error instanceof ApiError
            ? t(apiErrorKey(error.code))
            : t("errorCodes.unknown"),
        );
      }
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div>
      <header>
        <h1 className="font-heading text-2xl font-semibold tracking-tight">
          {t("account.security.title")}
        </h1>
        <p className="mt-1 text-muted-foreground">
          {t("account.security.subtitle")}
        </p>
      </header>

      <form
        className="mt-8 flex max-w-md flex-col gap-5"
        noValidate
        onSubmit={submit}
      >
        {formError ? <FormAlert description={formError} /> : null}

        <FormField
          label={t("account.security.currentPassword")}
          htmlFor="security-current"
          error={fieldErrors.current}
        >
          <Input
            id="security-current"
            type="password"
            autoComplete="current-password"
            value={current}
            onChange={(event) => setCurrent(event.target.value)}
          />
        </FormField>

        <FormField
          label={t("account.security.newPassword")}
          htmlFor="security-next"
          hint={t("account.security.passwordHint")}
          error={fieldErrors.next}
        >
          <Input
            id="security-next"
            type="password"
            autoComplete="new-password"
            value={next}
            onChange={(event) => setNext(event.target.value)}
          />
        </FormField>

        <FormField
          label={t("account.security.confirmPassword")}
          htmlFor="security-confirm"
          error={fieldErrors.confirm}
        >
          <Input
            id="security-confirm"
            type="password"
            autoComplete="new-password"
            value={confirm}
            onChange={(event) => setConfirm(event.target.value)}
          />
        </FormField>

        <Button
          type="submit"
          size="lg"
          className="self-start"
          disabled={submitting}
        >
          {t("account.security.submit")}
        </Button>
      </form>
    </div>
  );
}

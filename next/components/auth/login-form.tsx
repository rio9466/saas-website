"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { Link, useRouter } from "@/i18n/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useAuth } from "@/lib/auth";
import { ApiError, apiErrorKey } from "@/lib/api-error";
import { safeRedirect, stripLocalePrefix } from "@/lib/format";
import { FormAlert } from "./form-alert";
import { FormField } from "./form-field";

interface LoginFormProps {
  redirect: string | null;
  initialIdentifier: string;
  registered: boolean;
  passwordReset: boolean;
  passwordChanged: boolean;
  registrationEnabled: boolean;
  usernameLoginEnabled: boolean;
  emailLoginEnabled: boolean;
}

export function LoginForm({
  redirect,
  initialIdentifier,
  registered,
  passwordReset,
  passwordChanged,
  registrationEnabled,
  usernameLoginEnabled,
  emailLoginEnabled,
}: LoginFormProps) {
  const t = useTranslations();
  const router = useRouter();
  const { login, resendVerification } = useAuth();

  const [identifier, setIdentifier] = useState(initialIdentifier);
  const [password, setPassword] = useState("");
  const [fieldErrors, setFieldErrors] = useState<{
    identifier?: string;
    password?: string;
  }>({});
  const [formError, setFormError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [unverified, setUnverified] = useState(false);
  const [resendEmail, setResendEmail] = useState("");
  const [resending, setResending] = useState(false);
  const [resent, setResent] = useState(false);

  const loginDisabled = !usernameLoginEnabled && !emailLoginEnabled;
  const identifierHint =
    usernameLoginEnabled && emailLoginEnabled
      ? t("auth.login.hintBoth")
      : emailLoginEnabled
        ? t("auth.login.hintEmail")
        : t("auth.login.hintUsername");

  function validate(): boolean {
    const next: typeof fieldErrors = {};
    if (!identifier.trim()) {
      next.identifier = t("auth.login.errors.identifierRequired");
    }
    if (!password) {
      next.password = t("auth.login.errors.passwordRequired");
    }
    setFieldErrors(next);
    return Object.keys(next).length === 0;
  }

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setFormError("");
    setUnverified(false);
    if (!validate()) {
      return;
    }
    setSubmitting(true);
    try {
      await login({ identifier: identifier.trim(), password });
      const target = safeRedirect(redirect);
      router.push(target ? stripLocalePrefix(target) : "/account");
    } catch (error) {
      if (error instanceof ApiError && error.code === 40011) {
        setUnverified(true);
        setResendEmail(identifier.includes("@") ? identifier.trim() : "");
      } else if (error instanceof ApiError && error.code === 10001) {
        setFormError(t("auth.login.errors.validation"));
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

  async function resend() {
    if (!resendEmail.trim()) {
      return;
    }
    setFormError("");
    setResending(true);
    try {
      await resendVerification(resendEmail.trim());
      setResent(true);
    } catch (error) {
      setFormError(
        error instanceof ApiError
          ? t(apiErrorKey(error.code))
          : t("errorCodes.unknown"),
      );
    } finally {
      setResending(false);
    }
  }

  return (
    <form className="mt-8 flex flex-col gap-5" noValidate onSubmit={submit}>
      {registered ? (
        <FormAlert tone="success" description={t("auth.login.registered")} />
      ) : null}
      {passwordReset ? (
        <FormAlert tone="success" description={t("auth.login.passwordReset")} />
      ) : null}
      {passwordChanged ? (
        <FormAlert tone="success" description={t("auth.login.passwordChanged")} />
      ) : null}

      {loginDisabled ? (
        <FormAlert tone="warning" description={t("errorCodes.40012")} />
      ) : null}

      {formError ? <FormAlert description={formError} /> : null}

      {unverified ? (
        <FormAlert
          tone="warning"
          title={t("auth.login.unverifiedTitle")}
          description={t("auth.login.unverifiedDescription")}
          action={
            <div className="flex w-full flex-col gap-2 sm:flex-row sm:items-center">
              <Input
                type="email"
                className="w-full sm:max-w-xs"
                placeholder={t("auth.login.fields.email")}
                value={resendEmail}
                onChange={(event) => setResendEmail(event.target.value)}
              />
              <Button
                type="button"
                variant="outline"
                disabled={resending || resent || !resendEmail.trim()}
                onClick={resend}
              >
                {resent ? t("auth.login.resent") : t("auth.login.resend")}
              </Button>
            </div>
          }
        />
      ) : null}

      <FormField
        label={t("auth.login.fields.identifier")}
        htmlFor="login-identifier"
        hint={identifierHint}
        error={fieldErrors.identifier}
      >
        <Input
          id="login-identifier"
          autoComplete="username"
          value={identifier}
          onChange={(event) => setIdentifier(event.target.value)}
        />
      </FormField>

      <FormField
        label={t("auth.login.fields.password")}
        htmlFor="login-password"
        error={fieldErrors.password}
      >
        <Input
          id="login-password"
          type="password"
          autoComplete="current-password"
          value={password}
          onChange={(event) => setPassword(event.target.value)}
        />
      </FormField>

      <div className="flex items-center justify-between">
        <Link
          href="/forgot-password"
          className="text-sm text-primary hover:underline"
        >
          {t("auth.login.forgot")}
        </Link>
      </div>

      <Button
        type="submit"
        size="lg"
        className="w-full"
        disabled={submitting || loginDisabled}
      >
        {t("auth.login.submit")}
      </Button>

      {registrationEnabled ? (
        <p className="text-center text-sm text-muted-foreground">
          {t("auth.login.noAccount")}{" "}
          <Link href="/register" className="text-primary hover:underline">
            {t("auth.login.signUp")}
          </Link>
        </p>
      ) : null}
    </form>
  );
}

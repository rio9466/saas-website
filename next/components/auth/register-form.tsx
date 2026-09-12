"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { Link, useRouter } from "@/i18n/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useAuth } from "@/lib/auth";
import { ApiError, apiErrorKey } from "@/lib/api-error";
import { EMAIL_PATTERN, byteLength } from "@/lib/format";
import { FormAlert } from "./form-alert";
import { FormField } from "./form-field";

export function RegisterForm({
  registrationEnabled,
}: {
  registrationEnabled: boolean;
}) {
  const t = useTranslations();
  const router = useRouter();
  const { register, resendVerification } = useAuth();

  const [username, setUsername] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [fieldErrors, setFieldErrors] = useState<{
    username?: string;
    email?: string;
    password?: string;
    confirm?: string;
  }>({});
  const [formError, setFormError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [status, setStatus] = useState<"form" | "pending">("form");
  const [createdEmail, setCreatedEmail] = useState("");
  const [resending, setResending] = useState(false);
  const [resent, setResent] = useState(false);

  function validate(): boolean {
    const next: typeof fieldErrors = {};
    if (!username.trim()) {
      next.username = t("auth.register.errors.usernameRequired");
    } else if (username.trim().length < 3 || username.trim().length > 64) {
      next.username = t("auth.register.errors.usernameLength");
    }
    if (!email.trim()) {
      next.email = t("auth.register.errors.emailRequired");
    } else if (!EMAIL_PATTERN.test(email.trim())) {
      next.email = t("auth.register.errors.emailInvalid");
    }
    const passwordBytes = byteLength(password);
    if (!password) {
      next.password = t("auth.register.errors.passwordRequired");
    } else if (passwordBytes < 8 || passwordBytes > 72) {
      next.password = t("auth.register.errors.passwordLength");
    }
    if (!confirm) {
      next.confirm = t("auth.register.errors.confirmRequired");
    } else if (confirm !== password) {
      next.confirm = t("auth.register.errors.confirmMismatch");
    }
    setFieldErrors(next);
    return Object.keys(next).length === 0;
  }

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setFormError("");
    if (!validate()) {
      return;
    }
    setSubmitting(true);
    try {
      const user = await register({
        username: username.trim(),
        email: email.trim(),
        password,
      });
      if (user.status === "pending_verification") {
        setCreatedEmail(user.email);
        setStatus("pending");
      } else {
        router.push({
          pathname: "/login",
          query: { identifier: user.username, registered: "1" },
        });
      }
    } catch (error) {
      setFormError(
        error instanceof ApiError && error.code === 10001
          ? t("auth.register.errors.validation")
          : error instanceof ApiError
            ? t(apiErrorKey(error.code))
            : t("errorCodes.unknown"),
      );
    } finally {
      setSubmitting(false);
    }
  }

  async function resend() {
    setFormError("");
    setResending(true);
    try {
      await resendVerification(createdEmail);
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

  if (!registrationEnabled) {
    return (
      <div className="mt-8">
        <FormAlert
          tone="warning"
          title={t("auth.register.closedTitle")}
          description={t("auth.register.closedDescription")}
        />
      </div>
    );
  }

  if (status === "pending") {
    return (
      <div className="mt-8">
        <FormAlert
          tone="success"
          title={t("auth.register.pendingTitle")}
          description={t("auth.register.pendingDescription", {
            email: createdEmail,
          })}
          action={
            <div className="flex flex-wrap items-center gap-2">
              <Button
                type="button"
                variant="outline"
                disabled={resending || resent}
                onClick={resend}
              >
                {resent ? t("auth.register.resent") : t("auth.register.resend")}
              </Button>
              <Link
                href="/login"
                className="text-sm text-primary hover:underline"
              >
                {t("auth.register.backToLogin")}
              </Link>
            </div>
          }
        />
      </div>
    );
  }

  return (
    <form className="mt-8 flex flex-col gap-5" noValidate onSubmit={submit}>
      {formError ? <FormAlert description={formError} /> : null}

      <FormField
        label={t("auth.register.fields.username")}
        htmlFor="register-username"
        error={fieldErrors.username}
      >
        <Input
          id="register-username"
          autoComplete="username"
          value={username}
          onChange={(event) => setUsername(event.target.value)}
        />
      </FormField>

      <FormField
        label={t("auth.register.fields.email")}
        htmlFor="register-email"
        error={fieldErrors.email}
      >
        <Input
          id="register-email"
          type="email"
          autoComplete="email"
          value={email}
          onChange={(event) => setEmail(event.target.value)}
        />
      </FormField>

      <FormField
        label={t("auth.register.fields.password")}
        htmlFor="register-password"
        hint={t("auth.register.fields.passwordHint")}
        error={fieldErrors.password}
      >
        <Input
          id="register-password"
          type="password"
          autoComplete="new-password"
          value={password}
          onChange={(event) => setPassword(event.target.value)}
        />
      </FormField>

      <FormField
        label={t("auth.register.fields.confirmPassword")}
        htmlFor="register-confirm"
        error={fieldErrors.confirm}
      >
        <Input
          id="register-confirm"
          type="password"
          autoComplete="new-password"
          value={confirm}
          onChange={(event) => setConfirm(event.target.value)}
        />
      </FormField>

      <Button
        type="submit"
        size="lg"
        className="w-full"
        disabled={submitting}
      >
        {t("auth.register.submit")}
      </Button>

      <p className="text-center text-sm text-muted-foreground">
        {t("auth.register.haveAccount")}{" "}
        <Link href="/login" className="text-primary hover:underline">
          {t("auth.register.login")}
        </Link>
      </p>
    </form>
  );
}

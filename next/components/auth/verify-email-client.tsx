"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslations } from "next-intl";
import { Link } from "@/i18n/navigation";
import { Button } from "@/components/ui/button";
import { AppLoading } from "@/components/app-loading";
import { useAuth } from "@/lib/auth";
import { ApiError, apiErrorKey } from "@/lib/api-error";
import { FormAlert } from "./form-alert";

export function VerifyEmailClient({
  email,
  token,
}: {
  email: string;
  token: string;
}) {
  const t = useTranslations();
  const { verifyEmail, resendVerification } = useAuth();

  const [state, setState] = useState<"verifying" | "success" | "error">(
    "verifying",
  );
  const [errorKey, setErrorKey] = useState<string>("");
  const [resending, setResending] = useState(false);
  const [resent, setResent] = useState(false);
  const startedRef = useRef(false);

  const verify = useCallback(async () => {
    if (!email || !token) {
      setState("error");
      setErrorKey("auth.verifyEmail.missing");
      return;
    }
    try {
      await verifyEmail(email, token);
      setState("success");
    } catch (error) {
      setState("error");
      setErrorKey(
        error instanceof ApiError
          ? error.code === 40016
            ? "auth.verifyEmail.invalidDescription"
            : apiErrorKey(error.code)
          : "errorCodes.unknown",
      );
    }
  }, [email, token, verifyEmail]);

  useEffect(() => {
    if (startedRef.current) {
      return;
    }
    startedRef.current = true;
    void verify();
  }, [verify]);

  async function resend() {
    setResending(true);
    setErrorKey("");
    try {
      await resendVerification(email);
      setResent(true);
    } catch (error) {
      setErrorKey(
        error instanceof ApiError
          ? apiErrorKey(error.code)
          : "errorCodes.unknown",
      );
    } finally {
      setResending(false);
    }
  }

  if (state === "verifying") {
    return <AppLoading />;
  }

  if (state === "success") {
    return (
      <div className="mt-8">
        <FormAlert
          tone="success"
          title={t("auth.verifyEmail.successTitle")}
          description={t("auth.verifyEmail.successDescription")}
          action={
            <Link
              href="/login"
              className="text-sm text-primary hover:underline"
            >
              {t("auth.verifyEmail.goToLogin")}
            </Link>
          }
        />
      </div>
    );
  }

  return (
    <div className="mt-8">
      <FormAlert
        title={t("auth.verifyEmail.invalidTitle")}
        description={t(errorKey)}
        action={
          <div className="flex flex-wrap items-center gap-2">
            {email ? (
              <Button
                type="button"
                variant="outline"
                disabled={resending || resent}
                onClick={resend}
              >
                {resent
                  ? t("auth.verifyEmail.resent")
                  : t("auth.verifyEmail.resend")}
              </Button>
            ) : null}
            <Link
              href="/login"
              className="text-sm text-primary hover:underline"
            >
              {t("auth.verifyEmail.goToLogin")}
            </Link>
          </div>
        }
      />
    </div>
  );
}

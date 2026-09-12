"use client";

import { useTranslations } from "next-intl";
import { LogOutIcon, UserRoundIcon } from "lucide-react";
import { Link, useRouter } from "@/i18n/navigation";
import { useAuth } from "@/lib/auth";
import { Button, buttonVariants } from "@/components/ui/button";

/**
 * Header auth links. Server-rendered as logged-out (the access token lives in
 * memory and is only restored by the account guard), so this never causes a
 * hydration mismatch; it updates after login or a session restore.
 */
export function AuthNav() {
  const t = useTranslations("header");
  const router = useRouter();
  const { user, isAuthenticated, logout } = useAuth();

  async function onLogout() {
    await logout();
    router.push("/");
  }

  if (!isAuthenticated || !user) {
    return (
      <>
        <Link
          href="/login"
          className={buttonVariants({ variant: "ghost", size: "sm" })}
        >
          {t("login")}
        </Link>
        <Link href="/register" className={buttonVariants({ size: "sm" })}>
          {t("register")}
        </Link>
      </>
    );
  }

  return (
    <>
      <Link
        href="/account"
        className={buttonVariants({ variant: "ghost", size: "sm" })}
      >
        <UserRoundIcon />
        <span className="hidden sm:inline">
          {user.nickname || user.username || t("console")}
        </span>
      </Link>
      <Button type="button" variant="ghost" size="sm" onClick={onLogout}>
        <LogOutIcon />
        <span className="hidden sm:inline">{t("logout")}</span>
      </Button>
    </>
  );
}

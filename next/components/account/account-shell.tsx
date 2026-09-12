"use client";

import { useEffect, useRef, useState } from "react";
import { useTranslations } from "next-intl";
import {
  CoinsIcon,
  LayoutDashboardIcon,
  LogOutIcon,
  ShieldCheckIcon,
  UserRoundIcon,
} from "lucide-react";
import { cn } from "cn";
import { Link, usePathname, useRouter } from "@/i18n/navigation";
import { useAuth } from "@/lib/auth";
import { AppLoading } from "@/components/app-loading";
import { Button, buttonVariants } from "@/components/ui/button";

const NAV_ITEMS = [
  { path: "/account", key: "account.nav.overview", icon: LayoutDashboardIcon },
  { path: "/account/points", key: "account.nav.points", icon: CoinsIcon },
  { path: "/account/security", key: "account.nav.security", icon: ShieldCheckIcon },
  { path: "/account/profile", key: "account.nav.profile", icon: UserRoundIcon },
] as const;

/**
 * Client-side guard for `/account/**` (contract §2). SSR never has the refresh
 * cookie (`Path=/api/v1/auth`), so the first render is a loading shell identical
 * on the server and during hydration; the session is restored after mount and
 * unauthenticated visitors are sent to `/[locale]/login?redirect=<path>`.
 */
export function AccountShell({ children }: { children: React.ReactNode }) {
  const t = useTranslations();
  const router = useRouter();
  const pathname = usePathname();
  const { user, isAuthenticated, restoreSession, logout } = useAuth();

  const initial = useRef({ isAuthenticated, restoreSession, router, pathname });
  const [ready, setReady] = useState(isAuthenticated);
  const [loggingOut, setLoggingOut] = useState(false);

  useEffect(() => {
    if (initial.current.isAuthenticated) {
      return;
    }
    let active = true;
    const { restoreSession: restore, router: navigate, pathname: from } =
      initial.current;
    (async () => {
      const restored = await restore();
      if (!active) {
        return;
      }
      if (!restored) {
        navigate.replace({ pathname: "/login", query: { redirect: from } });
        return;
      }
      setReady(true);
    })();
    return () => {
      active = false;
    };
  }, []);

  async function onLogout() {
    setLoggingOut(true);
    await logout();
    router.push("/");
  }

  if (!ready || !user) {
    return (
      <div className="mx-auto w-full max-w-6xl px-4 py-16">
        <AppLoading />
      </div>
    );
  }

  function isActive(path: string): boolean {
    return path === "/account"
      ? pathname === "/account"
      : pathname.startsWith(path);
  }

  return (
    <div className="mx-auto grid w-full max-w-6xl gap-8 px-4 py-10 lg:grid-cols-[16rem_minmax(0,1fr)]">
      <aside className="flex flex-col gap-4">
        <div className="rounded-lg border p-4">
          <p className="text-xs uppercase tracking-wide text-muted-foreground">
            {t("account.signedInAs")}
          </p>
          <p className="mt-1 truncate font-medium">
            {user.nickname || user.username}
          </p>
          <p className="truncate text-sm text-muted-foreground">{user.email}</p>
        </div>

        <nav className="flex gap-1 overflow-x-auto lg:flex-col lg:overflow-visible">
          {NAV_ITEMS.map((item) => {
            const active = isActive(item.path);
            const Icon = item.icon;
            return (
              <Link
                key={item.path}
                href={item.path}
                className={cn(
                  buttonVariants({
                    variant: active ? "secondary" : "ghost",
                  }),
                  "shrink-0 justify-start",
                )}
                aria-current={active ? "page" : undefined}
              >
                <Icon />
                {t(item.key)}
              </Link>
            );
          })}
        </nav>

        <Button
          type="button"
          variant="outline"
          className="justify-start"
          disabled={loggingOut}
          onClick={onLogout}
        >
          <LogOutIcon />
          {t("account.logout")}
        </Button>
      </aside>

      <section>{children}</section>
    </div>
  );
}

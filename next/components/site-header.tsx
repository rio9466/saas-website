"use client";

import { useTranslations } from "next-intl";
import { MenuIcon } from "lucide-react";
import { Link } from "@/i18n/navigation";
import { Button, buttonVariants } from "@/components/ui/button";
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { LocaleSwitcher } from "@/components/locale-switcher";
import { AuthNav } from "@/components/auth-nav";
import { SiteLogo } from "@/components/site-logo";
import { ThemeToggle } from "@/components/theme-toggle";
import type { SiteSettings } from "@/lib/site-settings";

// Static nav for the foundation; backend-driven navigation arrives with the
// public pages (NEXT-02).
const NAV_ITEMS = [
  { href: "/", key: "nav.home" },
  { href: "/features", key: "nav.features" },
  { href: "/pricing", key: "nav.pricing" },
  { href: "/docs", key: "nav.docs" },
  { href: "/contact", key: "nav.contact" },
] as const;

export function SiteHeader({ settings }: { settings: SiteSettings }) {
  const t = useTranslations();

  return (
    <header className="sticky top-0 z-40 border-b bg-background/95 backdrop-blur supports-backdrop-filter:bg-background/60">
      <div className="mx-auto flex h-14 w-full max-w-6xl items-center gap-2 px-4">
        <Link
          href="/"
          className="flex items-center"
          aria-label={settings.site_name}
        >
          <SiteLogo settings={settings} />
        </Link>

        <nav
          className="ml-4 hidden items-center gap-1 md:flex"
          aria-label={t("footer.navigation")}
        >
          {NAV_ITEMS.map((item) => (
            <Link
              key={item.href}
              href={item.href}
              className={buttonVariants({ variant: "ghost", size: "sm" })}
            >
              {t(item.key)}
            </Link>
          ))}
        </nav>

        <div className="ml-auto flex items-center gap-1">
          <LocaleSwitcher locales={settings.locales} />
          <ThemeToggle />

          <AuthNav />

          <Sheet>
            <SheetTrigger
              render={
                <Button
                  className="md:hidden"
                  variant="ghost"
                  size="icon"
                  aria-label={t("header.openMenu")}
                />
              }
            >
              <MenuIcon />
            </SheetTrigger>
            <SheetContent side="right">
              <SheetHeader>
                <SheetTitle>{settings.site_name}</SheetTitle>
              </SheetHeader>
              <nav className="flex flex-col gap-1 px-4">
                {NAV_ITEMS.map((item) => (
                  <SheetClose
                    key={item.href}
                    render={
                      <Link
                        href={item.href}
                        className={buttonVariants({
                          variant: "ghost",
                          className: "justify-start",
                        })}
                      />
                    }
                  >
                    {t(item.key)}
                  </SheetClose>
                ))}
              </nav>
            </SheetContent>
          </Sheet>
        </div>
      </div>
    </header>
  );
}

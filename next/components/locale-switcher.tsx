"use client";

import { useLocale, useTranslations } from "next-intl";
import { CheckIcon, LanguagesIcon } from "lucide-react";
import { usePathname, useRouter } from "@/i18n/navigation";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import type { AppLocale } from "@/i18n/routing";
import type { SiteLocale } from "@/lib/site-settings";

/**
 * Language switcher. Uses the next-intl navigation APIs so the locale prefix
 * is kept and the `NEXT_LOCALE` cookie is set on switch (persisted across
 * reloads and new visits).
 */
export function LocaleSwitcher({ locales }: { locales: SiteLocale[] }) {
  const t = useTranslations("locale");
  const locale = useLocale();
  const pathname = usePathname();
  const router = useRouter();

  const current = locales.find((item) => item.code === locale);

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button
            variant="ghost"
            size="sm"
            aria-label={t("switchLabel")}
            title={t("switchLabel")}
          />
        }
      >
        <LanguagesIcon />
        <span className="hidden sm:inline">{current?.label ?? locale}</span>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        {locales.map((item) => (
          <DropdownMenuItem
            key={item.code}
            onClick={() => {
              if (item.code !== locale) {
                router.replace(pathname, { locale: item.code as AppLocale });
              }
            }}
          >
            <span>{item.label}</span>
            {item.code === locale ? <CheckIcon className="ml-auto" /> : null}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

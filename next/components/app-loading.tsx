"use client";

import { useTranslations } from "next-intl";
import { Loader2Icon } from "lucide-react";

/** Shared loading shell, mirrored on the server and during hydration. */
export function AppLoading() {
  const t = useTranslations("common");

  return (
    <div
      className="flex items-center justify-center gap-2 py-12 text-sm text-muted-foreground"
      role="status"
      aria-live="polite"
    >
      <Loader2Icon className="size-5 animate-spin" />
      <span>{t("loading")}</span>
    </div>
  );
}

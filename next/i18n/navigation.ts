import { createNavigation } from "next-intl/navigation";
import { routing } from "./routing";

/**
 * Locale-aware navigation. Every in-site link must use these exports instead
 * of `next/link` / `next/navigation`, otherwise the locale prefix is dropped
 * (contract §7.3) — the Next.js equivalent of Nuxt's `localePath`.
 */
export const { Link, redirect, usePathname, useRouter, getPathname } =
  createNavigation(routing);

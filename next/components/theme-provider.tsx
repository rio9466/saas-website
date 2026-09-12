"use client";

import type { ComponentProps } from "react";
import { ThemeProvider as NextThemesProvider } from "next-themes";

/**
 * next-themes wrapper. It persists the choice (localStorage by default) and
 * applies the `dark` class; `<html suppressHydrationWarning>` in the layout
 * keeps the first paint from flashing before the script runs.
 */
export function ThemeProvider({
  children,
  ...props
}: ComponentProps<typeof NextThemesProvider>) {
  return <NextThemesProvider {...props}>{children}</NextThemesProvider>;
}

import type { SiteSettings } from "@/lib/site-settings";

/** Site logo when one is configured, otherwise the site name as text. */
export function SiteLogo({
  settings,
  className,
}: {
  settings: SiteSettings;
  className?: string;
}) {
  if (settings.logo_url) {
    return (
      // Logos come from the backend at an arbitrary URL, so `next/image` is
      // not used (it would require per-domain remote patterns).
      // eslint-disable-next-line @next/next/no-img-element
      <img
        src={settings.logo_url}
        alt={settings.site_name}
        className={className ?? "h-7 w-auto"}
      />
    );
  }

  return (
    <span className={className ?? "text-base font-semibold"}>
      {settings.site_name}
    </span>
  );
}

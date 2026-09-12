import { useTranslations } from "next-intl";
import { Link } from "@/i18n/navigation";
import { SiteLogo } from "@/components/site-logo";
import type { SiteSettings } from "@/lib/site-settings";

const NAV_ITEMS = [
  { href: "/", key: "nav.home" },
  { href: "/features", key: "nav.features" },
  { href: "/pricing", key: "nav.pricing" },
  { href: "/docs", key: "nav.docs" },
  { href: "/about", key: "nav.about" },
  { href: "/contact", key: "nav.contact" },
] as const;

export function SiteFooter({ settings }: { settings: SiteSettings }) {
  const t = useTranslations();
  const year = new Date().getFullYear();

  return (
    <footer className="border-t bg-muted/30">
      <div className="mx-auto grid w-full max-w-6xl gap-8 px-4 py-10 sm:grid-cols-2 lg:grid-cols-4">
        <div className="space-y-3">
          <SiteLogo settings={settings} />
          {settings.tagline ? (
            <p className="max-w-xs text-sm text-muted-foreground">
              {settings.tagline}
            </p>
          ) : null}
        </div>

        <nav aria-label={t("footer.navigation")} className="space-y-2 text-sm">
          <p className="font-medium">{t("footer.navigation")}</p>
          <ul className="space-y-1">
            {NAV_ITEMS.map((item) => (
              <li key={item.href}>
                <Link
                  href={item.href}
                  className="text-muted-foreground hover:text-foreground"
                >
                  {t(item.key)}
                </Link>
              </li>
            ))}
          </ul>
        </nav>

        <div className="space-y-2 text-sm">
          <p className="font-medium">{t("footer.contact")}</p>
          <ul className="space-y-1 text-muted-foreground">
            {settings.contact_email ? (
              <li>
                <a href={`mailto:${settings.contact_email}`}>
                  {settings.contact_email}
                </a>
              </li>
            ) : null}
            {settings.contact_phone ? (
              <li>
                <a href={`tel:${settings.contact_phone}`}>
                  {settings.contact_phone}
                </a>
              </li>
            ) : null}
            {settings.contact_address ? (
              <li>{settings.contact_address}</li>
            ) : null}
          </ul>
        </div>

        {settings.social_links.length ? (
          <div className="space-y-2 text-sm">
            <p className="font-medium">{t("footer.followUs")}</p>
            <ul className="flex flex-wrap gap-3 text-muted-foreground">
              {settings.social_links.map((link) => (
                <li key={`${link.platform}-${link.url}`}>
                  <a
                    href={link.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="hover:text-foreground"
                  >
                    {link.platform}
                  </a>
                </li>
              ))}
            </ul>
          </div>
        ) : null}
      </div>

      <div className="border-t">
        <div className="mx-auto flex w-full max-w-6xl flex-col items-center gap-1 px-4 py-4 text-center text-xs text-muted-foreground">
          <p>
            {settings.footer_text ||
              `© ${year} ${settings.site_name}. ${t("footer.rights")}`}
          </p>
          {settings.icp_record ? (
            <p>
              {t("footer.icp")}: {settings.icp_record}
            </p>
          ) : null}
        </div>
      </div>
    </footer>
  );
}

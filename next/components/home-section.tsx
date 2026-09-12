import { buttonVariants } from "@/components/ui/button";
import { FeatureCard } from "@/components/feature-card";
import { LocalizedLink } from "@/components/localized-link";
import type { FeatureItem, HomeSection as HomeSectionData } from "@/lib/public-content";

/** Section types this template knows how to render (contract §4.3). */
export const KNOWN_SECTION_TYPES = new Set([
  "hero",
  "features",
  "screenshot",
  "stats",
  "cta",
]);

function text(data: Record<string, unknown>, key: string): string {
  const value = data?.[key];
  return typeof value === "string" ? value : "";
}

function array(
  data: Record<string, unknown>,
  key: string,
): Array<Record<string, unknown>> {
  const value = data?.[key];
  return Array.isArray(value) ? (value as Array<Record<string, unknown>>) : [];
}

function entryText(entry: Record<string, unknown>, key: string): string {
  const value = entry[key];
  return typeof value === "string" ? value : "";
}

/**
 * Renders one home section. Unknown section types are filtered out by the
 * caller, so this only handles the known set.
 */
export function HomeSection({
  section,
  features,
}: {
  section: HomeSectionData;
  features: FeatureItem[];
}) {
  const data = section.data ?? {};
  const title = text(data, "title");
  const subtitle =
    text(data, "subtitle") || text(data, "description") || text(data, "body");
  const image = text(data, "image_url") || text(data, "image");
  const primaryLabel = text(data, "primary_cta_label") || text(data, "cta_label");
  const primaryUrl =
    text(data, "primary_cta_url") || text(data, "cta_url") || "/register";
  const secondaryLabel = text(data, "secondary_cta_label");
  const secondaryUrl = text(data, "secondary_cta_url") || "/contact";
  const stats = array(data, "items").length
    ? array(data, "items")
    : array(data, "stats");

  if (section.type === "hero") {
    return (
      <section className="mx-auto w-full max-w-5xl px-4 py-20 text-center sm:py-28">
        <h1 className="font-heading text-4xl font-semibold tracking-tight sm:text-5xl">
          {title}
        </h1>
        {subtitle ? (
          <p className="mx-auto mt-4 max-w-2xl text-lg text-muted-foreground">
            {subtitle}
          </p>
        ) : null}
        {primaryLabel || secondaryLabel ? (
          <div className="mt-8 flex flex-wrap items-center justify-center gap-3">
            {primaryLabel ? (
              <LocalizedLink
                href={primaryUrl}
                className={buttonVariants({ size: "lg" })}
              >
                {primaryLabel}
              </LocalizedLink>
            ) : null}
            {secondaryLabel ? (
              <LocalizedLink
                href={secondaryUrl}
                className={buttonVariants({ variant: "outline", size: "lg" })}
              >
                {secondaryLabel}
              </LocalizedLink>
            ) : null}
          </div>
        ) : null}
        {image ? (
          // Hero images come from the backend at an arbitrary URL.
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={image}
            alt={title}
            className="mx-auto mt-12 w-full max-w-4xl rounded-xl"
          />
        ) : null}
      </section>
    );
  }

  if (section.type === "features") {
    return (
      <section className="mx-auto w-full max-w-6xl px-4 py-16">
        {title ? (
          <h2 className="text-center font-heading text-3xl font-semibold">
            {title}
          </h2>
        ) : null}
        {subtitle ? (
          <p className="mx-auto mt-3 max-w-2xl text-center text-muted-foreground">
            {subtitle}
          </p>
        ) : null}
        {features.length ? (
          <div className="mt-10 grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
            {features.map((feature) => (
              <FeatureCard key={feature.id} feature={feature} />
            ))}
          </div>
        ) : null}
      </section>
    );
  }

  if (section.type === "screenshot") {
    return (
      <section className="mx-auto w-full max-w-6xl px-4 py-16">
        {title ? (
          <h2 className="text-center font-heading text-3xl font-semibold">
            {title}
          </h2>
        ) : null}
        {subtitle ? (
          <p className="mx-auto mt-3 max-w-2xl text-center text-muted-foreground">
            {subtitle}
          </p>
        ) : null}
        {image ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={image}
            alt={title}
            loading="lazy"
            className="mx-auto mt-10 w-full rounded-xl border"
          />
        ) : null}
      </section>
    );
  }

  if (section.type === "stats") {
    return (
      <section className="mx-auto w-full max-w-6xl px-4 py-16">
        {title ? (
          <h2 className="text-center font-heading text-3xl font-semibold">
            {title}
          </h2>
        ) : null}
        {stats.length ? (
          <dl className="mt-10 grid gap-8 text-center sm:grid-cols-2 lg:grid-cols-4">
            {stats.map((stat, index) => (
              <div key={index} className="flex flex-col gap-1">
                <dt className="text-3xl font-bold text-primary">
                  {entryText(stat, "value")}
                </dt>
                <dd className="text-sm text-muted-foreground">
                  {entryText(stat, "label")}
                </dd>
              </div>
            ))}
          </dl>
        ) : null}
      </section>
    );
  }

  // cta
  return (
    <section className="mx-auto w-full max-w-6xl px-4 py-16">
      <div className="rounded-2xl border bg-muted/30 px-6 py-12 text-center">
        <h2 className="font-heading text-3xl font-semibold">{title}</h2>
        {subtitle ? (
          <p className="mx-auto mt-3 max-w-2xl text-muted-foreground">
            {subtitle}
          </p>
        ) : null}
        {primaryLabel ? (
          <LocalizedLink
            href={primaryUrl}
            className={buttonVariants({ size: "lg", className: "mt-8" })}
          >
            {primaryLabel}
          </LocalizedLink>
        ) : null}
      </div>
    </section>
  );
}

import { Blocks, ShieldCheck, Sparkles, Zap, type LucideIcon } from "lucide-react";
import { MarkdownContent } from "@/components/markdown-content";
import type { FeatureItem } from "@/lib/public-content";

/**
 * One feature card. Backend icons are stored as `i-lucide-<name>`; only a small
 * set is mapped (with a generic fallback) so the whole icon library is never
 * pulled into the bundle.
 */
const ICONS: Record<string, LucideIcon> = {
  zap: Zap,
  "shield-check": ShieldCheck,
  blocks: Blocks,
  sparkles: Sparkles,
};

export function FeatureCard({ feature }: { feature: FeatureItem }) {
  const Icon = ICONS[feature.icon.replace(/^i-lucide-/, "")] ?? Sparkles;

  return (
    <article className="flex flex-col gap-3 rounded-xl border bg-card p-6">
      <Icon className="size-6 text-primary" aria-hidden="true" />
      <h3 className="font-heading font-semibold">{feature.title}</h3>
      {feature.summary ? (
        <p className="text-sm text-muted-foreground">{feature.summary}</p>
      ) : null}
      {feature.image_url ? (
        // Feature images come from the backend at an arbitrary URL.
        // eslint-disable-next-line @next/next/no-img-element
        <img
          src={feature.image_url}
          alt={feature.title}
          loading="lazy"
          className="w-full rounded-lg"
        />
      ) : null}
      {feature.body_md ? <MarkdownContent content={feature.body_md} /> : null}
    </article>
  );
}

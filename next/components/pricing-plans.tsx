"use client";

import { useState } from "react";
import { useLocale, useTranslations } from "next-intl";
import { Check } from "lucide-react";
import { cn } from "cn";
import { Button, buttonVariants } from "@/components/ui/button";
import { LocalizedLink } from "@/components/localized-link";
import type { PricingPlan } from "@/lib/public-content";

type BillingPeriod = "monthly" | "yearly";

function formatPrice(
  plan: PricingPlan,
  period: BillingPeriod,
  fallbackCurrency: string,
  locale: string,
): string {
  const raw =
    period === "yearly" ? plan.yearly_price : plan.monthly_price;
  if (!raw) {
    return "—";
  }
  const amount = Number(raw);
  if (!Number.isFinite(amount)) {
    return raw;
  }
  const currency = plan.currency || fallbackCurrency || "USD";
  try {
    return new Intl.NumberFormat(locale, {
      style: "currency",
      currency,
      maximumFractionDigits: 2,
    }).format(amount);
  } catch {
    return `${amount} ${currency}`;
  }
}

export function PricingPlans({
  plans,
  currency,
}: {
  plans: PricingPlan[];
  currency: string;
}) {
  const t = useTranslations("pricing");
  const locale = useLocale();
  const [period, setPeriod] = useState<BillingPeriod>("monthly");

  const periods: BillingPeriod[] = ["monthly", "yearly"];

  return (
    <div>
      <div className="mt-8 flex justify-center">
        <div className="inline-flex rounded-lg border bg-muted/30 p-1">
          {periods.map((option) => (
            <Button
              key={option}
              type="button"
              size="sm"
              variant={period === option ? "default" : "ghost"}
              aria-pressed={period === option}
              onClick={() => setPeriod(option)}
            >
              {t(option)}
            </Button>
          ))}
        </div>
      </div>

      <div className="mt-10 grid items-stretch gap-6 lg:grid-cols-3">
        {plans.map((plan) => {
          const highlighted = plan.highlighted;
          return (
            <article
              key={plan.id}
              className={cn(
                "flex h-full flex-col gap-4 rounded-xl border bg-card p-6",
                highlighted && "border-primary shadow-lg",
              )}
            >
              <div className="flex items-start justify-between gap-2">
                <h3 className="font-heading text-lg font-semibold">
                  {plan.name}
                </h3>
                {highlighted ? (
                  <span className="rounded-full bg-primary px-2 py-0.5 text-xs font-medium text-primary-foreground">
                    {t("recommended")}
                  </span>
                ) : null}
              </div>

              {plan.description ? (
                <p className="text-sm text-muted-foreground">
                  {plan.description}
                </p>
              ) : null}

              <p className="flex items-baseline gap-1">
                <span className="text-3xl font-semibold">
                  {formatPrice(plan, period, currency, locale)}
                </span>
                <span className="text-sm text-muted-foreground">
                  {period === "yearly" ? t("perYear") : t("perMonth")}
                </span>
              </p>

              {plan.features?.length ? (
                <ul className="flex flex-1 flex-col gap-2 text-sm text-muted-foreground">
                  {plan.features.map((feature, index) => (
                    <li key={index} className="flex items-start gap-2">
                      <Check
                        className="mt-0.5 size-4 shrink-0 text-primary"
                        aria-hidden="true"
                      />
                      <span>{feature}</span>
                    </li>
                  ))}
                </ul>
              ) : null}

              {plan.cta_url || plan.cta_label ? (
                <LocalizedLink
                  href={plan.cta_url || "/contact"}
                  className={buttonVariants({
                    variant: highlighted ? "default" : "outline",
                    className: "w-full",
                  })}
                >
                  {plan.cta_label || t("contactUs")}
                </LocalizedLink>
              ) : null}
            </article>
          );
        })}
      </div>
    </div>
  );
}

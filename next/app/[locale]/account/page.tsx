"use client";

import { useLocale, useTranslations } from "next-intl";
import { useAuth } from "@/lib/auth";
import { formatDateTime, formatPoints } from "@/lib/format";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";

function DefinitionCard({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="rounded-lg border p-4">
      <dt className="text-sm text-muted-foreground">{label}</dt>
      <dd className="mt-1 font-medium">{children}</dd>
    </div>
  );
}

export default function AccountOverviewPage() {
  const t = useTranslations();
  const locale = useLocale();
  const { user } = useAuth();

  if (!user) {
    return null;
  }

  return (
    <div>
      <header>
        <h1 className="font-heading text-2xl font-semibold tracking-tight">
          {t("account.overview.title")}
        </h1>
        <p className="mt-1 text-muted-foreground">
          {t("account.overview.subtitle")}
        </p>
      </header>

      <div className="mt-8 flex items-center gap-4">
        <Avatar size="lg">
          {user.avatar_url ? (
            <AvatarImage src={user.avatar_url} alt={user.nickname || user.username} />
          ) : null}
          <AvatarFallback>{user.username.slice(0, 1).toUpperCase()}</AvatarFallback>
        </Avatar>
        <div>
          <p className="text-lg font-semibold">{user.nickname || user.username}</p>
          <p className="text-sm text-muted-foreground">{user.email}</p>
        </div>
      </div>

      <dl className="mt-8 grid gap-4 sm:grid-cols-2">
        <DefinitionCard label={t("account.overview.username")}>
          {user.username}
        </DefinitionCard>
        <DefinitionCard label={t("account.overview.email")}>
          <span className="flex items-center gap-2">
            <span className="truncate">{user.email}</span>
            {user.status === "pending_verification" ? (
              <Badge variant="outline">{t("account.overview.unverified")}</Badge>
            ) : null}
          </span>
        </DefinitionCard>
        <DefinitionCard label={t("account.overview.level")}>
          {user.level?.name || t("account.overview.noLevel")}
        </DefinitionCard>
        <DefinitionCard label={t("account.overview.joinedAt")}>
          {formatDateTime(user.created_at, locale)}
        </DefinitionCard>
        <DefinitionCard label={t("account.overview.pointsBalance")}>
          <span className="font-mono text-lg font-semibold">
            {formatPoints(user.points_balance)}
          </span>
        </DefinitionCard>
        <DefinitionCard label={t("account.overview.consumption")}>
          <span className="font-mono text-lg font-semibold">
            {formatPoints(user.consumption_points)}
          </span>
        </DefinitionCard>
      </dl>
    </div>
  );
}

"use client";

import { useEffect, useState } from "react";
import { useLocale, useTranslations } from "next-intl";
import { CoinsIcon } from "lucide-react";
import { api } from "@/lib/api";
import { ApiError, apiErrorKey } from "@/lib/api-error";
import type { PointTransactionPage } from "@/lib/auth";
import { useAuth } from "@/lib/auth";
import { formatDateTime, formatPoints, isZeroPoints } from "@/lib/format";
import { AppLoading } from "@/components/app-loading";
import { Button } from "@/components/ui/button";
import { FormAlert } from "@/components/auth/form-alert";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

const PAGE_SIZE = 20;

function deltaClass(delta: string): string {
  if (delta.trim().startsWith("-")) {
    return "text-destructive";
  }
  return isZeroPoints(delta) ? "text-muted-foreground" : "text-emerald-600 dark:text-emerald-400";
}

export default function AccountPointsPage() {
  const t = useTranslations();
  const locale = useLocale();
  const { user } = useAuth();

  const [page, setPage] = useState(1);
  const [reloadToken, setReloadToken] = useState(0);
  const [data, setData] = useState<PointTransactionPage | null>(null);
  const [pending, setPending] = useState(true);
  const [error, setError] = useState<unknown>(null);

  useEffect(() => {
    let active = true;
    (async () => {
      try {
        const result = await api.get<PointTransactionPage>(
          "/v1/me/point-transactions",
          { query: { page, page_size: PAGE_SIZE } },
        );
        if (active) {
          setData(result);
          setError(null);
        }
      } catch (err) {
        if (active) {
          setError(err);
        }
      } finally {
        if (active) {
          setPending(false);
        }
      }
    })();
    return () => {
      active = false;
    };
  }, [page, reloadToken]);

  function changePage(next: number) {
    setPending(true);
    setPage(next);
  }

  function reload() {
    setPending(true);
    setReloadToken((value) => value + 1);
  }

  const totalPages = data
    ? Math.max(1, Math.ceil(data.total / (data.page_size || PAGE_SIZE)))
    : 1;

  if (!user) {
    return null;
  }

  return (
    <div>
      <header>
        <h1 className="font-heading text-2xl font-semibold tracking-tight">
          {t("account.points.title")}
        </h1>
        <p className="mt-1 text-muted-foreground">
          {t("account.points.subtitle")}
        </p>
      </header>

      <div className="mt-8 grid gap-4 sm:grid-cols-3">
        <div className="rounded-lg border p-4">
          <p className="text-sm text-muted-foreground">
            {t("account.points.balance")}
          </p>
          <p className="mt-1 font-mono text-2xl font-semibold">
            {formatPoints(user.points_balance)}
          </p>
        </div>
        <div className="rounded-lg border p-4">
          <p className="text-sm text-muted-foreground">
            {t("account.points.consumption")}
          </p>
          <p className="mt-1 font-mono text-2xl font-semibold">
            {formatPoints(user.consumption_points)}
          </p>
        </div>
        <div className="rounded-lg border p-4">
          <p className="text-sm text-muted-foreground">
            {t("account.points.level")}
          </p>
          <p className="mt-1 text-2xl font-semibold">
            {user.level?.name || t("account.overview.noLevel")}
          </p>
        </div>
      </div>

      <div className="mt-8">
        {pending ? (
          <AppLoading />
        ) : error ? (
          <FormAlert
            description={
              error instanceof ApiError
                ? t(apiErrorKey(error.code))
                : t("errorCodes.unknown")
            }
            action={
              <Button type="button" variant="outline" onClick={reload}>
                {t("common.retry")}
              </Button>
            }
          />
        ) : !data?.items.length ? (
          <div className="flex flex-col items-center gap-2 rounded-lg border border-dashed py-12 text-muted-foreground">
            <CoinsIcon className="size-6" />
            <p className="text-sm">{t("account.points.empty")}</p>
          </div>
        ) : (
          <div className="overflow-hidden rounded-lg border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("account.points.table.time")}</TableHead>
                  <TableHead>{t("account.points.table.reason")}</TableHead>
                  <TableHead className="text-right">
                    {t("account.points.table.change")}
                  </TableHead>
                  <TableHead className="text-right">
                    {t("account.points.table.balance")}
                  </TableHead>
                  <TableHead className="text-right">
                    {t("account.points.table.consumptionChange")}
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data.items.map((item) => (
                  <TableRow key={item.id}>
                    <TableCell className="text-muted-foreground">
                      {formatDateTime(item.created_at, locale)}
                    </TableCell>
                    <TableCell>{item.reason || "—"}</TableCell>
                    <TableCell
                      className={`text-right font-mono ${deltaClass(item.points_delta)}`}
                    >
                      {formatPoints(item.points_delta)}
                    </TableCell>
                    <TableCell className="text-right font-mono">
                      {formatPoints(item.balance_after)}
                    </TableCell>
                    <TableCell className="text-right font-mono text-muted-foreground">
                      {formatPoints(item.consumption_delta)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}

        {data && data.total > 0 ? (
          <div className="mt-4 flex items-center justify-between">
            <p className="text-sm text-muted-foreground">
              {t("account.points.pageInfo", { page: data.page, total: totalPages })}
            </p>
            <div className="flex gap-2">
              <Button
                type="button"
                variant="outline"
                size="sm"
                disabled={page <= 1}
                onClick={() => changePage(Math.max(1, page - 1))}
              >
                {t("account.points.previous")}
              </Button>
              <Button
                type="button"
                variant="outline"
                size="sm"
                disabled={page >= totalPages}
                onClick={() => changePage(Math.min(totalPages, page + 1))}
              >
                {t("account.points.next")}
              </Button>
            </div>
          </div>
        ) : null}
      </div>
    </div>
  );
}

"use client";

import { useState, type FormEvent } from "react";
import { useLocale, useTranslations } from "next-intl";
import { CircleAlert, CircleCheck } from "lucide-react";
import { api } from "@/lib/api";
import { ApiError, apiErrorKey } from "@/lib/api-error";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";

interface ContactFields {
  name: string;
  email: string;
  company: string;
  message: string;
  consent: boolean;
  website: string;
}

const EMPTY: ContactFields = {
  name: "",
  email: "",
  company: "",
  message: "",
  consent: false,
  website: "",
};

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

/**
 * Contact form (contract §4.8). Every submission carries the active `locale`,
 * a hidden honeypot (`website`) and a `consent` flag. Field-level validation is
 * client-side; backend `10001` shows a generic validation message and `42901`
 * (and any other code) is mapped through `apiErrorKey`.
 */
export function ContactForm() {
  const t = useTranslations("contact");
  const tRoot = useTranslations();
  const locale = useLocale();

  const [form, setForm] = useState<ContactFields>(EMPTY);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [submitting, setSubmitting] = useState(false);
  const [submitted, setSubmitted] = useState(false);
  const [formError, setFormError] = useState("");

  function update<K extends keyof ContactFields>(
    key: K,
    value: ContactFields[K],
  ) {
    setForm((prev) => ({ ...prev, [key]: value }));
  }

  function validate(): boolean {
    const next: Record<string, string> = {
      name: form.name.trim() ? "" : t("errors.nameRequired"),
      email: !form.email.trim()
        ? t("errors.emailRequired")
        : EMAIL_PATTERN.test(form.email.trim())
          ? ""
          : t("errors.emailInvalid"),
      message: form.message.trim() ? "" : t("errors.messageRequired"),
      consent: form.consent ? "" : t("errors.consentRequired"),
    };
    setErrors(next);
    return !Object.values(next).some(Boolean);
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setFormError("");
    if (!validate()) {
      return;
    }
    setSubmitting(true);
    try {
      await api.post("/v1/public/contact", {
        name: form.name.trim(),
        email: form.email.trim(),
        company: form.company.trim(),
        message: form.message.trim(),
        locale,
        consent: form.consent,
        website: form.website,
      });
      setForm(EMPTY);
      setErrors({});
      setSubmitted(true);
    } catch (error) {
      setFormError(
        error instanceof ApiError
          ? error.code === 10001
            ? t("errors.validation")
            : tRoot(apiErrorKey(error.code))
          : tRoot("errorCodes.unknown"),
      );
    } finally {
      setSubmitting(false);
    }
  }

  if (submitted) {
    return (
      <div className="flex flex-col gap-3 rounded-xl border bg-card p-6">
        <div className="flex items-center gap-2">
          <CircleCheck className="size-5 text-primary" aria-hidden="true" />
          <p className="font-heading font-semibold">{t("successTitle")}</p>
        </div>
        <p className="text-sm text-muted-foreground">
          {t("successDescription")}
        </p>
        <Button
          type="button"
          variant="outline"
          className="self-start"
          onClick={() => {
            setSubmitted(false);
            setFormError("");
          }}
        >
          {t("sendAnother")}
        </Button>
      </div>
    );
  }

  return (
    <form className="flex flex-col gap-5" noValidate onSubmit={handleSubmit}>
      {formError ? (
        <div className="flex items-start gap-2 rounded-lg border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-destructive">
          <CircleAlert className="mt-0.5 size-4 shrink-0" aria-hidden="true" />
          <p>{formError}</p>
        </div>
      ) : null}

      {/* Honeypot: rendered and hidden off-screen (not display:none) so bots fill it. */}
      <div
        className="absolute -left-[9999px] top-auto h-px w-px overflow-hidden"
        aria-hidden="true"
      >
        <label htmlFor="contact-website">Website</label>
        <input
          id="contact-website"
          name="website"
          type="text"
          tabIndex={-1}
          autoComplete="off"
          value={form.website}
          onChange={(event) => update("website", event.target.value)}
        />
      </div>

      <div className="grid gap-5 sm:grid-cols-2">
        <div className="flex flex-col gap-2">
          <Label htmlFor="contact-name">{t("fields.name")}</Label>
          <Input
            id="contact-name"
            name="name"
            autoComplete="name"
            aria-invalid={Boolean(errors.name)}
            value={form.name}
            onChange={(event) => update("name", event.target.value)}
          />
          {errors.name ? (
            <p className="text-xs text-destructive">{errors.name}</p>
          ) : null}
        </div>

        <div className="flex flex-col gap-2">
          <Label htmlFor="contact-email">{t("fields.email")}</Label>
          <Input
            id="contact-email"
            name="email"
            type="email"
            autoComplete="email"
            aria-invalid={Boolean(errors.email)}
            value={form.email}
            onChange={(event) => update("email", event.target.value)}
          />
          {errors.email ? (
            <p className="text-xs text-destructive">{errors.email}</p>
          ) : null}
        </div>
      </div>

      <div className="flex flex-col gap-2">
        <Label htmlFor="contact-company">{t("fields.company")}</Label>
        <Input
          id="contact-company"
          name="company"
          autoComplete="organization"
          value={form.company}
          onChange={(event) => update("company", event.target.value)}
        />
      </div>

      <div className="flex flex-col gap-2">
        <Label htmlFor="contact-message">{t("fields.message")}</Label>
        <Textarea
          id="contact-message"
          name="message"
          rows={5}
          aria-invalid={Boolean(errors.message)}
          value={form.message}
          onChange={(event) => update("message", event.target.value)}
        />
        {errors.message ? (
          <p className="text-xs text-destructive">{errors.message}</p>
        ) : null}
      </div>

      <div className="flex flex-col gap-2">
        <div className="flex items-start gap-2">
          <input
            id="contact-consent"
            name="consent"
            type="checkbox"
            className="mt-0.5 size-4 rounded border-input accent-primary"
            checked={form.consent}
            onChange={(event) => update("consent", event.target.checked)}
          />
          <Label htmlFor="contact-consent" className="font-normal">
            {t("fields.consent")}
          </Label>
        </div>
        {errors.consent ? (
          <p className="text-xs text-destructive">{errors.consent}</p>
        ) : null}
      </div>

      <Button type="submit" size="lg" className="self-start" disabled={submitting}>
        {t("submit")}
      </Button>
    </form>
  );
}

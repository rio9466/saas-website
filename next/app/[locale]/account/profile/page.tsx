"use client";

import { useState } from "react";
import { useTranslations } from "next-intl";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useAuth, type UpdateProfilePayload } from "@/lib/auth";
import { ApiError, apiErrorKey } from "@/lib/api-error";
import { FormAlert } from "@/components/auth/form-alert";
import { FormField } from "@/components/auth/form-field";

export default function AccountProfilePage() {
  const t = useTranslations();
  const { user, updateProfile } = useAuth();

  const [nickname, setNickname] = useState(user?.nickname ?? "");
  const [avatarUrl, setAvatarUrl] = useState(user?.avatar_url ?? "");
  const [formError, setFormError] = useState("");
  const [saved, setSaved] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  if (!user) {
    return null;
  }

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!user) {
      return;
    }
    setFormError("");
    setSaved(false);

    const payload: UpdateProfilePayload = {};
    if (nickname.trim() !== user.nickname) {
      payload.nickname = nickname.trim();
    }
    if (avatarUrl.trim() !== user.avatar_url) {
      payload.avatar_url = avatarUrl.trim();
    }
    if (Object.keys(payload).length === 0) {
      setSaved(true);
      return;
    }

    setSubmitting(true);
    try {
      const updated = await updateProfile(payload);
      setNickname(updated.nickname);
      setAvatarUrl(updated.avatar_url);
      setSaved(true);
    } catch (error) {
      setFormError(
        error instanceof ApiError
          ? t(apiErrorKey(error.code))
          : t("errorCodes.unknown"),
      );
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div>
      <header>
        <h1 className="font-heading text-2xl font-semibold tracking-tight">
          {t("account.profile.title")}
        </h1>
        <p className="mt-1 text-muted-foreground">
          {t("account.profile.subtitle")}
        </p>
      </header>

      <form
        className="mt-8 flex max-w-md flex-col gap-5"
        noValidate
        onSubmit={submit}
      >
        {saved ? (
          <FormAlert tone="success" description={t("account.profile.saved")} />
        ) : null}
        {formError ? <FormAlert description={formError} /> : null}

        <FormField
          label={t("account.profile.nickname")}
          htmlFor="profile-nickname"
          hint={t("account.profile.nicknameHint")}
        >
          <Input
            id="profile-nickname"
            autoComplete="nickname"
            value={nickname}
            onChange={(event) => setNickname(event.target.value)}
          />
        </FormField>

        <FormField
          label={t("account.profile.avatarUrl")}
          htmlFor="profile-avatar"
        >
          <Input
            id="profile-avatar"
            placeholder="https://…"
            value={avatarUrl}
            onChange={(event) => setAvatarUrl(event.target.value)}
          />
        </FormField>

        <Button
          type="submit"
          size="lg"
          className="self-start"
          disabled={submitting}
        >
          {t("account.profile.submit")}
        </Button>
      </form>
    </div>
  );
}

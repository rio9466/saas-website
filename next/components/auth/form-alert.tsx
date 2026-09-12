import { CircleCheckIcon, TriangleAlertIcon } from "lucide-react";
import { cn } from "cn";
import {
  Alert,
  AlertDescription,
  AlertTitle,
} from "@/components/ui/alert";

type AlertTone = "error" | "warning" | "success";

const TONE_CLASSES: Record<AlertTone, string> = {
  error: "border-destructive/40 bg-destructive/5 text-destructive",
  warning:
    "border-amber-500/40 bg-amber-500/5 text-amber-700 dark:text-amber-400",
  success:
    "border-emerald-500/40 bg-emerald-500/5 text-emerald-700 dark:text-emerald-400",
};

/** Inline feedback alert with a tone. Used by the auth and account forms. */
export function FormAlert({
  tone = "error",
  title,
  description,
  action,
  className,
}: {
  tone?: AlertTone;
  title?: string;
  description?: string;
  action?: React.ReactNode;
  className?: string;
}) {
  const Icon = tone === "success" ? CircleCheckIcon : TriangleAlertIcon;
  const hasBody = Boolean(title || description);

  return (
    <Alert className={cn(TONE_CLASSES[tone], className)}>
      <Icon />
      {hasBody ? (
        <div className="flex flex-col gap-1">
          {title ? <AlertTitle>{title}</AlertTitle> : null}
          {description ? (
            <AlertDescription className="text-current">
              {description}
            </AlertDescription>
          ) : null}
        </div>
      ) : null}
      {action ? (
        <div className="col-span-full mt-2">{action}</div>
      ) : null}
    </Alert>
  );
}

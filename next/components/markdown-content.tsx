import { cn } from "cn";
import { renderMarkdown } from "@/lib/markdown";

/**
 * Renders backend `body_md`. The Markdown is parsed and sanitised by
 * `renderMarkdown` before it reaches `dangerouslySetInnerHTML`; raw HTML is
 * never rendered directly (contract §7.5).
 */
export function MarkdownContent({
  content = "",
  className,
}: {
  content?: string;
  className?: string;
}) {
  const { html } = renderMarkdown(content);
  if (!html) {
    return null;
  }

  return (
    <div
      className={cn(
        "text-sm leading-7 break-words",
        "[&>*:first-child]:mt-0",
        "[&_h1]:mt-6 [&_h1]:mb-3 [&_h1]:scroll-mt-20 [&_h1]:text-2xl [&_h1]:font-semibold",
        "[&_h2]:mt-6 [&_h2]:mb-3 [&_h2]:scroll-mt-20 [&_h2]:text-xl [&_h2]:font-semibold",
        "[&_h3]:mt-5 [&_h3]:mb-2 [&_h3]:scroll-mt-20 [&_h3]:text-lg [&_h3]:font-semibold",
        "[&_h4]:mt-4 [&_h4]:mb-2 [&_h4]:scroll-mt-20 [&_h4]:font-semibold",
        "[&_p]:my-3 [&_ul]:my-3 [&_ul]:list-disc [&_ul]:pl-6",
        "[&_ol]:my-3 [&_ol]:list-decimal [&_ol]:pl-6 [&_li]:my-1",
        "[&_a]:text-primary [&_a]:underline [&_a]:underline-offset-2",
        "[&_blockquote]:my-3 [&_blockquote]:border-l-2 [&_blockquote]:border-border [&_blockquote]:pl-4 [&_blockquote]:text-muted-foreground",
        "[&_code]:rounded [&_code]:bg-muted [&_code]:px-1 [&_code]:py-0.5 [&_code]:text-[0.85em]",
        "[&_pre]:my-3 [&_pre]:overflow-x-auto [&_pre]:rounded-lg [&_pre]:border [&_pre]:bg-muted [&_pre]:p-4",
        "[&_pre_code]:bg-transparent [&_pre_code]:p-0",
        "[&_hr]:my-6 [&_hr]:border-border",
        "[&_img]:my-3 [&_img]:max-w-full [&_img]:rounded-lg",
        "[&_table]:my-3 [&_table]:block [&_table]:w-full [&_table]:overflow-x-auto [&_table]:border-collapse",
        "[&_th]:border [&_th]:border-border [&_th]:px-3 [&_th]:py-2 [&_th]:text-left",
        "[&_td]:border [&_td]:border-border [&_td]:px-3 [&_td]:py-2 [&_td]:text-left",
        className,
      )}
      dangerouslySetInnerHTML={{ __html: html }}
    />
  );
}

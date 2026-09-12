import MarkdownIt from "markdown-it";
import sanitizeHtml from "sanitize-html";

/**
 * Renders backend `body_md` to safe HTML (contract §7.5), equivalent to the
 * `frontend/` `useMarkdown` composable.
 *
 * - Raw HTML inside the Markdown is escaped (`html: false`), so only the
 *   renderer's own tags can appear.
 * - The result is passed through a whitelist sanitizer as a second line of
 *   defence, so `dangerouslySetInnerHTML` never receives unsanitised content.
 * - External links get `target="_blank"` + `rel="noopener noreferrer"`.
 */

export interface MarkdownHeading {
  id: string;
  text: string;
  level: number;
}

export interface MarkdownResult {
  html: string;
  toc: MarkdownHeading[];
}

const ALLOWED_TAGS = [
  "h1", "h2", "h3", "h4", "h5", "h6",
  "p", "a", "ul", "ol", "li", "blockquote", "pre", "code",
  "strong", "em", "s", "del", "hr", "br",
  "table", "thead", "tbody", "tr", "th", "td",
  "img", "span",
];

const ALLOWED_ATTR = [
  "href", "title", "target", "rel", "src", "alt", "id", "align", "colspan", "rowspan",
];

const md = new MarkdownIt({
  html: false,
  linkify: true,
  breaks: false,
  typographer: false,
});

const defaultLinkOpen =
  md.renderer.rules.link_open ??
  ((tokens, idx, options, _env, self) => self.renderToken(tokens, idx, options));

function slugify(text: string): string {
  return (
    text
      .trim()
      .toLowerCase()
      .replace(/[^\w\u4e00-\u9fa5\s-]/g, "")
      .replace(/\s+/g, "-")
      .replace(/-+/g, "-")
      .replace(/^-|-$/g, "") || "section"
  );
}

/** Parses Markdown and returns sanitised HTML plus the extracted heading TOC. */
export function renderMarkdown(content: string): MarkdownResult {
  const toc: MarkdownHeading[] = [];
  const seen = new Map<string, number>();

  md.renderer.rules.heading_open = (tokens, idx, options, _env, self) => {
    const token = tokens[idx];
    if (!token) {
      return "";
    }
    const inline = tokens[idx + 1];
    const level = Number(token.tag.slice(1));
    const text =
      inline && inline.type === "inline"
        ? md.renderer.renderInlineAsText(inline.children ?? [], options, {})
        : "";

    let id = slugify(text);
    const count = seen.get(id) ?? 0;
    seen.set(id, count + 1);
    if (count > 0) {
      id = `${id}-${count}`;
    }

    token.attrSet("id", id);
    toc.push({ id, text, level });
    return self.renderToken(tokens, idx, options);
  };

  md.renderer.rules.link_open = (tokens, idx, options, env, self) => {
    const token = tokens[idx];
    const href = String(token?.attrGet("href") ?? "");
    if (token && /^https?:\/\//i.test(href)) {
      token.attrSet("target", "_blank");
      token.attrSet("rel", "noopener noreferrer");
    }
    return defaultLinkOpen(tokens, idx, options, env, self);
  };

  const raw = md.render(content || "");
  const html = sanitizeHtml(raw, {
    allowedTags: ALLOWED_TAGS,
    allowedAttributes: { "*": ALLOWED_ATTR },
    allowedSchemes: ["http", "https", "mailto", "tel"],
    allowProtocolRelative: false,
    disallowedTagsMode: "discard",
  });

  return { html, toc };
}

import type { MetadataRoute } from "next";

/** SEO baseline. The language-aware sitemap is added with NEXT-02. */
export default function robots(): MetadataRoute.Robots {
  return {
    rules: [
      {
        userAgent: "*",
        allow: "/",
        disallow: ["/api/"],
      },
    ],
  };
}

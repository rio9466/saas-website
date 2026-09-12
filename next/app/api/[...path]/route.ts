import { NextResponse } from "next/server";

/**
 * Same-origin API proxy (contract §2.1 / §7.1). The browser must only call
 * `/api/**` on this origin; this route handler forwards each request to the Go
 * service at `API_PROXY_TARGET`, which is read from the environment on every
 * request so it can change without a rebuild.
 *
 * Forwards query, method, body, Cookie and Origin/Referer, and relays the
 * backend's Set-Cookie (the refresh cookie is HttpOnly and SameSite=Strict,
 * so it only works same-origin). Hop-by-hop and length/encoding headers are
 * dropped because the runtime fetch handles body framing and decompression.
 */
export const runtime = "nodejs";
export const dynamic = "force-dynamic";

const STRIPPED_REQUEST_HEADERS = new Set([
  "host",
  "connection",
  "keep-alive",
  "proxy-authenticate",
  "proxy-authorization",
  "te",
  "trailer",
  "transfer-encoding",
  "upgrade",
  "content-length",
  "accept-encoding",
]);

const STRIPPED_RESPONSE_HEADERS = new Set([
  "content-encoding",
  "content-length",
  "transfer-encoding",
  "connection",
  "keep-alive",
]);

async function proxy(request: Request): Promise<Response> {
  const target = (process.env.API_PROXY_TARGET ?? "").replace(/\/+$/, "");
  if (!target) {
    return NextResponse.json(
      {
        code: 50001,
        message: "API_PROXY_TARGET is not configured",
        data: {},
        request_id: "",
      },
      { status: 503 },
    );
  }

  const url = new URL(request.url);
  const targetUrl = `${target}${url.pathname}${url.search}`;

  const headers = new Headers();
  request.headers.forEach((value, key) => {
    if (!STRIPPED_REQUEST_HEADERS.has(key.toLowerCase())) {
      headers.set(key, value);
    }
  });

  const hasBody = request.method !== "GET" && request.method !== "HEAD";
  const body = hasBody ? await request.arrayBuffer() : undefined;

  let upstream: Response;
  try {
    upstream = await fetch(targetUrl, { method: request.method, headers, body });
  } catch {
    return NextResponse.json(
      {
        code: 50001,
        message: "Upstream service is unavailable",
        data: {},
        request_id: "",
      },
      { status: 502 },
    );
  }

  const responseHeaders = new Headers();
  upstream.headers.forEach((value, key) => {
    if (!STRIPPED_RESPONSE_HEADERS.has(key.toLowerCase())) {
      responseHeaders.set(key, value);
    }
  });
  // `set-cookie` is not a single combinable value; relay each one verbatim.
  for (const cookie of upstream.headers.getSetCookie()) {
    responseHeaders.append("set-cookie", cookie);
  }

  return new Response(upstream.body, {
    status: upstream.status,
    statusText: upstream.statusText,
    headers: responseHeaders,
  });
}

export {
  proxy as GET,
  proxy as POST,
  proxy as PUT,
  proxy as PATCH,
  proxy as DELETE,
  proxy as HEAD,
  proxy as OPTIONS,
};

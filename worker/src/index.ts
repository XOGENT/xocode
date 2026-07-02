/**
 * xocode install-script server.
 *
 * Serves the install script at https://code.xogent.com/install from an R2
 * bucket, with a text/x-shellscript content-type so `curl … | bash` works and
 * a browser shows the script for review before running.
 */

export interface Env {
  ASSETS: R2Bucket;
}

const SCRIPT_KEY = "install.sh";

export default {
  async fetch(req: Request, env: Env): Promise<Response> {
    const url = new URL(req.url);

    if (req.method !== "GET" && req.method !== "HEAD") {
      return new Response("Method not allowed", { status: 405 });
    }
    if (url.pathname !== "/" && url.pathname !== "/install") {
      return new Response("Not found\n", { status: 404 });
    }

    const obj = await env.ASSETS.get(SCRIPT_KEY);
    if (!obj) {
      return new Response("install script not found\n", { status: 500 });
    }

    const headers = new Headers({
      "content-type": "text/x-shellscript; charset=utf-8",
      "cache-control": "public, max-age=300",
      "x-content-type-options": "nosniff",
    });
    const etag = obj.httpEtag;
    if (etag) headers.set("etag", etag);

    if (req.method === "HEAD") {
      return new Response(null, { headers });
    }
    return new Response(obj.body, { headers });
  },
};

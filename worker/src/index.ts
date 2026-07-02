/**
 * xocode install-script server.
 *
 * Serves the install script at https://code.xogent.com/install with a
 * text/x-shellscript content-type so `curl … | bash` works and a browser shows
 * the script for review before running.
 *
 * The script is bundled from ../../scripts/install.sh at deploy time (see the
 * [[rules]] Text loader in wrangler.toml), so scripts/install.sh stays the
 * single source of truth and each `wrangler deploy` ships the current version.
 */

// @ts-expect-error - bundled as text via the wrangler Text rule for *.sh
import installScript from "../../scripts/install.sh";

const SCRIPT: string = installScript as string;

export default {
  async fetch(req: Request): Promise<Response> {
    const url = new URL(req.url);

    if (req.method !== "GET" && req.method !== "HEAD") {
      return new Response("Method not allowed\n", { status: 405 });
    }
    if (url.pathname !== "/" && url.pathname !== "/install") {
      return new Response("Not found\n", { status: 404 });
    }

    const headers = new Headers({
      "content-type": "text/x-shellscript; charset=utf-8",
      "cache-control": "public, max-age=300",
      "x-content-type-options": "nosniff",
    });

    if (req.method === "HEAD") {
      return new Response(null, { headers });
    }
    return new Response(SCRIPT, { headers });
  },
};

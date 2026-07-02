# xocode install worker

Serves `install.sh` at `https://code.xogent.com/install` from the
`xocode-assets` R2 bucket, on the XOGENT Cloudflare account.

## One-time setup

```sh
# 1. Create the R2 bucket (once)
npx wrangler r2 bucket create xocode-assets

# 2. Upload the install script (CI does this on every release; run once now)
npx wrangler r2 object put xocode-assets/install.sh \
  --file ../scripts/install.sh --content-type "text/x-shellscript"

# 3. Deploy the worker
npx wrangler deploy

# 4. Bind the hostname. Either the `routes` entry in wrangler.toml (needs the
#    xogent.com zone active in this account), or add code.xogent.com as a
#    Worker Custom Domain in the dashboard (auto-creates DNS + cert).
```

## Verify

```sh
curl -I https://code.xogent.com/install   # 200, content-type: text/x-shellscript
curl https://code.xogent.com/install      # prints the script (review before piping)
```

## Keeping it in sync

`.github/workflows/release.yml` uploads `scripts/install.sh` to this bucket on
every non-prerelease tag, so the served script always matches the latest
release. Requires repo secrets `CLOUDFLARE_ACCOUNT_ID` and
`CLOUDFLARE_API_TOKEN` (scoped to R2 write).

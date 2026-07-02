# Serves code.xogent.com: the install script at /install and a landing page at /.
# Built by Coolify from the repo root as build context.
FROM caddy:2-alpine

COPY deploy/Caddyfile /etc/caddy/Caddyfile
COPY deploy/index.html /srv/index.html
# The install script is the single source of truth in scripts/install.sh.
COPY scripts/install.sh /srv/install

EXPOSE 80

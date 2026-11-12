#!/bin/sh
# Rewrites root-relative URLs in a built site so it works under a base path,
# such as a GitHub Pages project site at /taragen/. Taragen sites link from
# the domain root (/docs/...), which is what `taragen serve` uses locally.
#
#   tools/rebase-site.sh /taragen web/_public
set -eu
base="${1%/}"
dir="$2"
[ -n "$base" ] || exit 0
find "$dir" -name '*.html' -exec sed -E -i \
  -e "s#(href|src|action)=\"/([^/\"]|\")#\\1=\"$base/\\2#g" \
  -e "s#url=/#url=$base/#g" {} +
find "$dir" -name '*.css' -exec sed -E -i \
  -e "s#url\\((['\"]?)/([^/])#url(\\1$base/\\2#g" {} +

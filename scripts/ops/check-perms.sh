#!/bin/sh
set -e

ok=1
chk_dir() {
  p="$1"; mode_expected="$2"; owner_expected="$3"; group_expected="$4"
  [ -d "$p" ] || { echo "MISSING: $p"; ok=0; return; }
  st=$(stat -c %a:%U:%G "$p" 2>/dev/null || stat -f %Sp:%Su:%Sg "$p")
  case "$st" in
    *$mode_expected*:*$owner_expected*:*$group_expected*) :;;
    *) echo "BAD: $p has $st, expected $mode_expected:$owner_expected:$group_expected"; ok=0;;
  esac
}

chk_dir /etc/nos 750 nos nos
chk_dir /var/lib/nos 750 nos nos
chk_dir /var/lib/nos/state 750 nos nos

if [ -f /etc/nos/secret.key ]; then
  m=$(stat -c %a /etc/nos/secret.key 2>/dev/null || stat -f %Op /etc/nos/secret.key)
  case "$m" in
    600|0100600) :;;
    *) echo "BAD: /etc/nos/secret.key mode $m, expected 600"; ok=0;;
  esac
fi

exit $ok



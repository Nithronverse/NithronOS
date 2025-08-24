#!/bin/sh
set -euo pipefail

ROOT=${ROOT:-/}
ok=1

join() { printf "%s/%s\n" "$1" "$2" | sed 's#//#/#g'; }

chk_dir() {
  p="$1"; mode_expected="$2"; owner_expected="$3"; group_expected="$4"
  [ -d "$p" ] || { echo "MISSING: $p"; ok=0; return; }
  st=$(stat -c %a:%U:%G "$p" 2>/dev/null || stat -f %Sp:%Su:%Sg "$p" 2>/dev/null || echo "unknown:unknown:unknown")
  case "$st" in
    *$mode_expected*:*$owner_expected*:*$group_expected*) :;;
    *) echo "BAD: $p has $st, expected $mode_expected:$owner_expected:$group_expected"; ok=0;;
  esac
}

etc_nos=$(join "$ROOT" etc/nos)
var_lib_nos=$(join "$ROOT" var/lib/nos)
state_dir=$(join "$ROOT" var/lib/nos/state)
secret=$(join "$ROOT" etc/nos/secret.key)

chk_dir "$etc_nos" 750 nos nos
chk_dir "$var_lib_nos" 750 nos nos
chk_dir "$state_dir" 750 nos nos

if [ -f "$secret" ]; then
  m=$(stat -c %a "$secret" 2>/dev/null || stat -f %Op "$secret" 2>/dev/null || echo "0000")
  o=$(stat -c %U:%G "$secret" 2>/dev/null || stat -f %Su:%Sg "$secret" 2>/dev/null || echo ":")
  case "$m" in 600|0100600) :;; *) echo "BAD: $secret mode $m, expected 600"; ok=0;; esac
  case "$o" in nos:nos) :;; *) echo "BAD: $secret owner $o, expected nos:nos"; ok=0;; esac
fi

exit $ok



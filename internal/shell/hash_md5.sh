hash_md5() {
  target=${1:-/dev/stdin}
  if is_command md5sum; then
    sum=$(md5sum "$target" 2>/dev/null) || return 1
    echo "$sum" | cut -d ' ' -f 1
  elif is_command md5; then
    md5 -q "$target" 2>/dev/null
  else
    log_crit "hash_md5 unable to find command to compute md5 hash"
    return 1
  fi
}

hash_verify() {
  TARGET=$1
  checksums=$2
  want=$(extract_hash "${TARGET}" "${checksums}")
  if [ -z "$want" ]; then
    log_err "hash_verify unable to find checksum for '${TARGET}' in '${checksums}'"
    return 1
  fi
  got=$(hash_md5 "$TARGET")
  if [ "$want" != "$got" ]; then
    log_err "hash_verify checksum for '$TARGET' did not verify ${want} vs $got"
    return 1
  fi
}

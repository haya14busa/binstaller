hash_sha1() {
  TARGET=${1:-/dev/stdin}
  if is_command gsha1sum; then
    hash=$(gsha1sum "$TARGET") || return 1
    echo "$hash" | cut -d ' ' -f 1
  elif is_command sha1sum; then
    hash=$(sha1sum "$TARGET") || return 1
    echo "$hash" | cut -d ' ' -f 1
  elif is_command shasum; then
    hash=$(shasum -a 1 "$TARGET" 2>/dev/null) || return 1
    echo "$hash" | cut -d ' ' -f 1
  elif is_command openssl; then
    hash=$(openssl dgst -sha1 "$TARGET") || return 1
    echo "$hash" | cut -d ' ' -f 2
  else
    log_crit "hash_sha1 unable to find command to compute sha-1 hash"
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
  got=$(hash_sha1 "$TARGET")
  if [ "$want" != "$got" ]; then
    log_err "hash_verify checksum for '$TARGET' did not verify ${want} vs $got"
    return 1
  fi
}

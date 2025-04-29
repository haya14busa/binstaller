hash_sha256() {
  TARGET=${1:-/dev/stdin}
  if is_command gsha256sum; then
    hash=$(gsha256sum "$TARGET") || return 1
    echo "$hash" | cut -d ' ' -f 1
  elif is_command sha256sum; then
    hash=$(sha256sum "$TARGET") || return 1
    echo "$hash" | cut -d ' ' -f 1
  elif is_command shasum; then
    hash=$(shasum -a 256 "$TARGET" 2>/dev/null) || return 1
    echo "$hash" | cut -d ' ' -f 1
  elif is_command openssl; then
    hash=$(openssl dgst -sha256 "$TARGET") || return 1
    echo "$hash" | cut -d ' ' -f 2
  else
    log_crit "hash_sha256 unable to find command to compute sha-256 hash"
    return 1
  fi
}

hash_verify() {
  hash_verify_internal "$1" "$2" hash_sha256
}

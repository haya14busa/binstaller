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

hash_compute() {
  hash_sha1 "$1"
}

hash_verify() {
  hash_verify_internal "$1" "$2" hash_sha1
}

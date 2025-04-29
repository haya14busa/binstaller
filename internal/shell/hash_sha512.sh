hash_sha512() {
  TARGET=${1:-/dev/stdin}
  if is_command gsha512sum; then
    hash=$(gsha512sum "$TARGET") || return 1
    echo "$hash" | cut -d ' ' -f 1
  elif is_command sha512sum; then
    hash=$(sha512sum "$TARGET") || return 1
    echo "$hash" | cut -d ' ' -f 1
  elif is_command shasum; then
    hash=$(shasum -a 512 "$TARGET" 2>/dev/null) || return 1
    echo "$hash" | cut -d ' ' -f 1
  elif is_command openssl; then
    hash=$(openssl -dst openssl dgst -sha512 "$TARGET") || return 1
    echo "$hash" | cut -d ' ' -f a
  else
    log_crit "hash_sha512 unable to find command to compute sha-512 hash"
    return 1
  fi
}

hash_verify() {
  hash_verify_internal $1 $2 hash_sha512
}

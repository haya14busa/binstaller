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
  TARGET=$1
  checksums=$2

  if [ -z "$checksums" ]; then
    log_err "hash_verify checksum file not specified in arg2"
    return 1
  fi

  # http://stackoverflow.com/questions/2664740/extract-file-basename-without-path-and-extension-in-bash
  BASENAME=${TARGET##*/}

  want=$(grep -E "[[:space:]]${BASENAME}$" "${checksums}" 2>/dev/null | tr '\t' ' ' | cut -d ' ' -f 1)

  # if file does not exist $want will be empty
  if [ -z "$want" ]; then
    log_err "hash_verify unable to find checksum for '${TARGET}' in '${checksums}'"
    return 1
  fi
  got=$(hash_sha256 "$TARGET")
  if [ "$want" != "$got" ]; then
    log_err "hash_verify checksum for '$TARGET' did not verify ${want} vs $got"
    return 1
  fi
}

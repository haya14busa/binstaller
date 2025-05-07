untar() {
  tarball=$1
  strip_components=${2:-0} # default 0
  case "${tarball}" in
  *.tar.gz | *.tgz) tar --no-same-owner -xzf "${tarball}" --strip-components "${strip_components}" ;;
  *.tar.xz) tar --no-same-owner -xJf "${tarball}" --strip-components "${strip_components}" ;;
  *.tar.bz2) tar --no-same-owner -xjf "${tarball}" --strip-components "${strip_components}" ;;
  *.tar) tar --no-same-owner -xf "${tarball}" --strip-components "${strip_components}" ;;
  *.gz) gunzip "${tarball}" ;;
  *.zip)
    # unzip doesn't have a standard --strip-components
    # Workaround: extract to a subdir and move contents up if stripping
    if [ "$strip_components" -gt 0 ]; then
      extract_dir=$(basename "${tarball%.zip}")_extracted
      unzip -q "${tarball}" -d "${extract_dir}"
      # Move contents of the *first* directory found inside extract_dir up
      # This assumes wrap_in_directory=true convention
      first_subdir=$(find "${extract_dir}" -mindepth 1 -maxdepth 1 -type d -print -quit)
      if [ -n "$first_subdir" ]; then
        # Move all contents (* includes hidden files)
        mv "${first_subdir}"/* .
        # Optionally remove the now-empty subdir and the extract_dir
        rmdir "${first_subdir}"
        rmdir "${extract_dir}"
      else
        log_warn "Could not find subdirectory in zip to strip components from ${extract_dir}"
        # Files are extracted in current dir anyway, proceed
      fi
    else
      unzip -q "${tarball}"
    fi
    ;;
  *)
    log_err "untar unknown archive format for ${tarball}"
    return 1
    ;;
  esac
}

extract_hash() {
  TARGET=$1
  checksums=$2
  if [ -z "$checksums" ]; then
    log_err "extract_hash checksum file not specified in arg2"
    return 1
  fi
  BASENAME=${TARGET##*/}
  grep -E "([[:space:]]|/|\*)${BASENAME}$" "${checksums}" 2>/dev/null | tr '\t' ' ' | cut -d ' ' -f 1
}

hash_verify_internal() {
  TARGET_PATH=$1
  SUMFILE=$2
  HASH_FUNC=$3
  if [ -z "${SUMFILE}" ]; then
    log_err "hash_verify_internal checksum file not specified in arg2"
    return 1
  fi
  if [ -z "${HASH_FUNC}" ]; then
    log_err "hash_verify_internal hash func not specified in arg3"
    return 1
  fi
  got=$($HASH_FUNC "$TARGET_PATH")
  if [ -z "${got}" ]; then
    log_err "failed to calculate hash: ${TARGET_PATH}"
    return 1
  fi
  # 1) “hash-only” line?
  if grep -i -E "^${got}[[:space:]]*$" "$SUMFILE" >/dev/null 2>&1; then
    return 0
  fi
  # 2) Check hash & file name match
  want=$(extract_hash "${TARGET_PATH}" "${SUMFILE}")
  if [ "$want" != "$got" ]; then
    log_err "hash_verify checksum for '$TARGET_PATH' did not verify ${want} vs ${got}"
    return 1
  fi
}

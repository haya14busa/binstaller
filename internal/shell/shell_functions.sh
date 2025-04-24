untar() {
  tarball=$1
  strip_components=${2:-0} # Second argument is strip_components, default 0
  strip_components_flag=""
  if [ "$strip_components" -gt 0 ]; then
   strip_components_flag="--strip-components=${strip_components}"
  fi

  case "${tarball}" in
    *.tar.gz | *.tgz) tar --no-same-owner -xzf "${tarball}" ${strip_components_flag} ;;
    *.tar.xz) tar --no-same-owner -xJf "${tarball}" ${strip_components_flag} ;;
    *.tar) tar --no-same-owner -xf "${tarball}" ${strip_components_flag} ;;
    *.zip)
       # unzip doesn't have a standard --strip-components
       # Workaround: extract to a subdir and move contents up if stripping
       if [ "$strip_components" -gt 0 ]; then
          extract_dir=$(basename "${tarball%.zip}")_extracted
          unzip "${tarball}" -d "${extract_dir}"
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
          unzip "${tarball}"
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

  # http://stackoverflow.com/questions/2664740/extract-file-basename-without-path-and-extension-in-bash
  BASENAME=${TARGET##*/}

  echo "$(grep -E "([[:space:]]|/)${BASENAME}$" "${checksums}" 2>/dev/null | tr '\t' ' ' | cut -d ' ' -f 1)"

  # if file does not exist $want will be empty
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

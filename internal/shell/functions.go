package shell

// shellFunctions contains the library of POSIX shell functions.
// Adapted from https://github.com/client9/shlib
const shellFunctions = `
# --- Shell Functions (from shlib) ---
cat /dev/null <<EOF
------------------------------------------------------------------------
https://github.com/client9/shlib - portable posix shell functions
Public domain - http://unlicense.org
https://github.com/client9/shlib/blob/master/LICENSE.md
but credit (and pull requests) appreciated.
------------------------------------------------------------------------
EOF
is_command() {
  command -v "$1" >/dev/null
}
echoerr() {
  echo "$@" 1>&2
}
log_prefix() {
  # Default implementation - can be overridden by main script
  echo "$0"
}
_logp=6
log_set_priority() {
  _logp="$1"
}
log_priority() {
  if test -z "$1"; then
    echo "$_logp"
    return
  fi
  [ "$1" -le "$_logp" ]
}
log_tag() {
  case $1 in
    0) echo "emerg" ;;
    1) echo "alert" ;;
    2) echo "crit" ;;
    3) echo "err" ;;
    4) echo "warning" ;;
    5) echo "notice" ;;
    6) echo "info" ;;
    7) echo "debug" ;;
    *) echo "$1" ;;
  esac
}
log_debug() {
  log_priority 7 || return 0
  echoerr "$(log_prefix)" "$(log_tag 7)" "$@"
}
log_info() {
  log_priority 6 || return 0
  echoerr "$(log_prefix)" "$(log_tag 6)" "$@"
}
log_err() {
  log_priority 3 || return 0
  echoerr "$(log_prefix)" "$(log_tag 3)" "$@"
}
log_crit() {
  log_priority 2 || return 0
  echoerr "$(log_prefix)" "$(log_tag 2)" "$@"
}
uname_os() {
  os=$(uname -s | tr '[:upper:]' '[:lower:]')
  case "$os" in
    cygwin_nt*) os="windows" ;;
    mingw*) os="windows" ;;
    msys_nt*) os="windows" ;;
  esac
  echo "$os"
}
uname_arch() {
  arch=$(uname -m)
  case $arch in
    x86_64) arch="amd64" ;;
    x86) arch="386" ;;
    i686) arch="386" ;;
    i386) arch="386" ;;
    aarch64) arch="arm64" ;;
    armv5*) arch="armv5" ;;
    armv6*) arch="armv6" ;;
    armv7*) arch="armv7" ;;
  esac
  echo ${arch}
}
uname_os_check() {
  os=$(uname_os)
  case "$os" in
    darwin) return 0 ;;
    dragonfly) return 0 ;;
    freebsd) return 0 ;;
    linux) return 0 ;;
    android) return 0 ;;
    nacl) return 0 ;;
    netbsd) return 0 ;;
    openbsd) return 0 ;;
    plan9) return 0 ;;
    solaris) return 0 ;;
    windows) return 0 ;;
  esac
  log_crit "uname_os_check '$(uname -s)' got converted to '$os' which is not a GOOS value. Please file bug at https://github.com/client9/shlib"
  return 1
}
uname_arch_check() {
  arch=$(uname_arch)
  case "$arch" in
    386) return 0 ;;
    amd64) return 0 ;;
    arm64) return 0 ;;
    armv5) return 0 ;;
    armv6) return 0 ;;
    armv7) return 0 ;;
    ppc64) return 0 ;;
    ppc64le) return 0 ;;
    mips) return 0 ;;
    mipsle) return 0 ;;
    mips64) return 0 ;;
    mips64le) return 0 ;;
    s390x) return 0 ;;
    amd64p32) return 0 ;;
  esac
  log_crit "uname_arch_check '$(uname -m)' got converted to '$arch' which is not a GOARCH value.  Please file bug report at https://github.com/client9/shlib"
  return 1
}
untar() {
  tarball=$1
  strip_components=${2:-0} # Second argument is strip_components, default 0
  strip_components_flag=""
  if [ "$strip_components" -gt 0 ]; then
   strip_components_flag="--strip-components=${strip_components}"
  fi

  case "${tarball}" in
    *.tar.gz | *.tgz) tar --no-same-owner -xzf "${tarball}" ${strip_components_flag} ;;
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
http_download_curl() {
  local_file=$1
  source_url=$2
  header=$3
  if [ -z "$header" ]; then
    code=$(curl -w '%{http_code}' -sL -o "$local_file" "$source_url")
  else
    code=$(curl -w '%{http_code}' -sL -H "$header" -o "$local_file" "$source_url")
  fi
  if [ "$code" != "200" ]; then
    log_debug "http_download_curl received HTTP status $code"
    # Attempt to print error from body if available
    if [ -f "$local_file" ]; then
       log_debug "Error body:"
       cat "$local_file" 1>&2
    fi
    rm -f "$local_file" # Remove potentially incomplete file
    return 1
  fi
  return 0
}
http_download_wget() {
  local_file=$1
  source_url=$2
  header=$3
  if [ -z "$header" ]; then
    wget --quiet --output-document="$local_file" "$source_url"
  else
    wget --quiet --header="$header" --output-document="$local_file" "$source_url"
  fi
}
http_download() {
  log_debug "http_download $2"
  if is_command curl; then
    http_download_curl "$@"
    return $? # Propagate exit code
  elif is_command wget; then
    http_download_wget "$@"
    return $? # Propagate exit code
  fi
  log_crit "http_download unable to find wget or curl"
  return 1
}
http_copy() {
  tmp=$(mktemp)
  http_download "${tmp}" "$1" "$2" || { rm -f "${tmp}"; return 1; } # Cleanup on failure
  body=$(cat "$tmp")
  rm -f "${tmp}"
  echo "$body"
}
github_release() {
  owner_repo=$1
  version=$2
  # Function to get release tag from GitHub API
  # Needs implementation using http_copy and JSON parsing (e.g., grep/sed or jq if available)
  # Handle "latest" tag specifically.
  log_debug "Fetching release tag for ${owner_repo}, version ${version}"
  # Simplified: return version for now, assuming tag is passed correctly
  if [ -z "$version" ] || [ "$version" = "latest" ]; then
     # Use GitHub API to find the latest release tag
     api_url="https://api.github.com/repos/${owner_repo}/releases/latest"
     # Use Authorization header if GITHUB_TOKEN is set
     auth_header=""
     if [ -n "$GITHUB_TOKEN" ]; then
       # Ensure header format is correct
       auth_header="Authorization: token $GITHUB_TOKEN"
       log_debug "Using GITHUB_TOKEN for API request"
     fi
     # Pass header correctly to http_copy -> http_download
     json=$(http_copy "$api_url" "$auth_header")
     if [ -z "$json" ]; then
        log_err "Failed to fetch latest release info from GitHub API (url: $api_url)"
        return 1
     fi
     # Basic parsing with sed (requires POSIX ERE support)
     # Handle potential spaces around colon, use [^"]* to match tag name
     tag=$(echo "$json" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p')
     if [ -z "$tag" ]; then
        log_err "Could not parse tag_name from GitHub API response (url: $api_url)"
        log_debug "Response body: $json"
        return 1
     fi
     echo "$tag"
     return 0
  else
    # Assume version is a valid tag
    echo "$version"
    return 0
  fi
}
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
    # Note: openssl output format can vary. Adjust parsing as needed.
    # Example for some versions: "SHA256(filename)= hash"
    hash=$(openssl dgst -sha256 "$TARGET") || return 1
    # Try to extract the hash, assuming it's the last field
    echo "$hash" | awk '{print $NF}'
  else
    log_crit "hash_sha256 unable to find command to compute sha-256 hash"
    return 1
  fi
}
hash_sha256_verify() {
  TARGET=$1
  checksums=$2 # This is the checksum file path
  # TODO: Adapt for embedded checksums from spec.Checksums.EmbeddedChecksums
  if [ -z "$checksums" ]; then
    log_err "hash_sha256_verify checksum file not specified"
    return 1
  fi
  BASENAME=${TARGET##*/}
  # Use awk for potentially more robust parsing than grep | cut
  # Match format "hash<space(s)>filename" or "hash<space(s)>*filename" (for BSD sum)
  want=$(awk -v filename="${BASENAME}" '{ if ($2 == filename || $2 == "*" filename) { print $1; exit } }' "${checksums}")

  if [ -z "$want" ]; then
    log_err "hash_sha256_verify unable to find checksum for '${BASENAME}' in '${checksums}'"
    return 1
  fi
  got=$(hash_sha256 "$TARGET")
  if [ "$want" != "$got" ]; then
    log_err "hash_sha256_verify checksum for '$TARGET' did not verify."
    log_err "Expected: $want"
    log_err "Got:      $got"
    return 1
  fi
  log_info "Checksum verified for ${TARGET}"
}
cat /dev/null <<EOF
------------------------------------------------------------------------
End of functions from https://github.com/client9/shlib
------------------------------------------------------------------------
EOF
`

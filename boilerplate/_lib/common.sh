err() {
  echo "$@" >&2
  exit 1
}

## osdk_version BINARY
#
# Print the version of the specified operator-sdk BINARY
osdk_version() {
    local osdk=$1
    # `operator-sdk version` output looks like
    #       operator-sdk version: v0.8.2, commit: 28bd2b0d4fd25aa68e15d928ae09d3c18c3b51da
    # or
    #       operator-sdk version: "v0.16.0", commit: "55f1446c5f472e7d8e308dcdf36d0d7fc44fc4fd", go version: "go1.13.8 linux/amd64"
    # Peel out the version number, accounting for the optional quotes.
    $osdk version | sed 's/operator-sdk version: "*\([^,"]*\)"*,.*/\1/'
}

# Only used for error messages
_lib=${BASH_SOURCE##*/}

# When this lib is sourced (which is what it's designed for), $0 is the
# script that did the sourcing.
SOURCER=$(realpath $0)
[[ -n "$SOURCER" ]] || err "$_lib couldn't discover where it was sourced from"

HERE=${SOURCER%/*}
[[ -n "$HERE" ]] || err "$_lib failed to discover the dirname of sourcer at $SOURCER"

REPO_ROOT=$(git rev-parse --show-toplevel)
[[ -n "$REPO_ROOT" ]] || err "$_lib couldn't discover the repo root"

CONVENTION_ROOT=$REPO_ROOT/boilerplate
[[ -d "$CONVENTION_ROOT" ]] || err "$CONVENTION_ROOT: not a directory"

# Were we sourced from within a convention?
if [[ "$HERE" == "$CONVENTION_ROOT/"* ]]; then
  # Okay, figure out the name of the convention
  CONVENTION_NAME=${HERE#$CONVENTION_ROOT/}
  # If we got here, we really expected to be able to identify the
  # convention name.
  [[ -n "$CONVENTION_NAME" ]] || err "$_lib couldn't discover the name of the sourcing convention"
fi


# Set SED variable
if LANG=C sed --help 2>&1 | grep -q GNU; then
  SED="sed"
elif command -v gsed &>/dev/null; then
  SED="gsed"
else
  echo "Failed to find GNU sed as sed or gsed. If you are on Mac: brew install gnu-sed." >&2
  exit 1
fi

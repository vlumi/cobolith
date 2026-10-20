#!/bin/sh
# Fetches NIST's COBOL-85 test suite, CCVS85, into .cache/ccvs85/ unless it
# is there: from the Internet Archive's copy of NIST's address, which no
# longer serves it, checked against the hash Guix pins it by, and
# uncompressed to newcob.val.
set -eu

dir=.cache/ccvs85
url=http://web.archive.org/web/2010id_/http://www.itl.nist.gov/div897/ctg/suites/newcob.val.Z
sha256=1e9a92ddbd5d730cbeb764281f7810c22b18e0163985b09675393ab22bbd61f9

[ -f "$dir/newcob.val" ] && exit 0
mkdir -p "$dir"
curl --silent --show-error --location --fail --output "$dir/newcob.val.Z" "$url"
if command -v sha256sum >/dev/null; then
    actual=$(sha256sum "$dir/newcob.val.Z" | cut -d' ' -f1)
else
    actual=$(shasum -a 256 "$dir/newcob.val.Z" | cut -d' ' -f1)
fi
if [ "$actual" != "$sha256" ]; then
    echo "newcob.val.Z: sha256 $actual, want $sha256" >&2
    rm "$dir/newcob.val.Z"
    exit 1
fi
gzip -dc "$dir/newcob.val.Z" > "$dir/newcob.val"

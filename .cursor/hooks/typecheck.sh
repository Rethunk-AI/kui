#!/bin/bash
# Run Go build at session end to surface compile errors.
# Always exits 0 — surfaces errors in the log for the next session.
# Cursor expects valid JSON on stdout; redirect build output to stderr.
cat > /dev/null
(go build -o /dev/null ./cmd/... 2>&1) >&2
echo '{}'
exit 0

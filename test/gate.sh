#!/bin/sh
set -eu

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

cat >"$tmpdir/case-1.yaml" <<'EOF'
case_id: case-1
skill: demo-skill
input:
  prompt: alpha
EOF

cat >"$tmpdir/case-2.yaml" <<'EOF'
case_id: case-2
skill: demo-skill
input:
  prompt: beta
EOF

run_json="$tmpdir/run.json"
score_json="$tmpdir/score.json"
pass_policy="$tmpdir/pass-policy.yaml"
fail_policy="$tmpdir/fail-policy.yaml"

./seh run --skill demo-skill --cases "$tmpdir" --out "$run_json"
./seh score --run "$run_json" --out "$score_json"

cat >"$pass_policy" <<'EOF'
min_score: 0
min_success_rate: 1
max_p95_latency: 100000
max_avg_tokens: 10
EOF

cat >"$fail_policy" <<'EOF'
min_score: 1
min_success_rate: 1
max_p95_latency: 0
max_avg_tokens: 0
EOF

./seh gate --report "$score_json" --policy "$pass_policy"

if ./seh gate --report "$score_json" --policy "$fail_policy"; then
  echo "expected gate rejection to exit 1"
  exit 1
fi

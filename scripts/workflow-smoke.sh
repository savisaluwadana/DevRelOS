#!/bin/sh
# Walks the DevRelOS workflow documented in docs/WORKFLOW.md section 3 against a
# running API, asserting that each surface actually functions and that the three
# loops connect: discover -> engage -> measure.
#
# This exercises product behaviour, complementing the auth/session/secret smoke
# test in .github/workflows/ci.yml. It found real defects that unit tests could
# not: an always-empty Command Center high-fit panel, campaign reports that never
# populated their item list, and database errors surfacing as HTTP 500.
#
# Usage:
#   DEVRELOS_API_BASE=http://127.0.0.1:8080 DEVRELOS_API_TOKEN=... sh scripts/workflow-smoke.sh
#
# Requires: curl, jq. Expects an empty project: it asserts on first-time
# conversions that are guarded by uniqueness constraints.
set -u
API="${DEVRELOS_API_BASE:-http://127.0.0.1:8080}/api/v1"
TOKEN="${DEVRELOS_API_TOKEN:-}"

# Wrap curl once rather than interpolating a header: the value contains spaces
# and cannot survive command substitution unsplit. Works in single-operator
# local mode (no token) and with DEVRELOS_REQUIRE_AUTH=true alike.
if [ -n "$TOKEN" ]; then
  ccurl() { curl -sS -H "Authorization: Bearer $TOKEN" "$@"; }
else
  ccurl() { curl -sS "$@"; }
fi
PASS=0; FAIL=0
say()  { printf '%s\n' "$1"; }
ok()   { PASS=$((PASS+1)); printf '  PASS  %s\n' "$1"; }
bad()  { FAIL=$((FAIL+1)); printf '  FAIL  %s -- %s\n' "$1" "$2"; }

post() { ccurl -w '\n%{http_code}' -H 'Content-Type: application/json' -d "$2" "$API$1"; }
get()  { ccurl -w '\n%{http_code}' "$API$1"; }
body() { printf '%s' "$1" | sed '$d'; }
code() { printf '%s' "$1" | tail -1; }
jqf()  { printf '%s' "$1" | jq -r "$2" 2>/dev/null; }

step() { # name, method-output, expected-code
  c=$(code "$2")
  if [ "$c" = "$3" ]; then ok "$1"; else bad "$1" "HTTP $c: $(body "$2" | head -c 200)"; fi
}

say "== 1. Talk Library =="
R=$(post /talks '{"title":"Scaling Kubernetes without a platform team","abstract":"Lessons learned","description":"d","level":"intermediate","durationMinutes":30,"topics":["kubernetes","platform engineering"],"status":"ready","demoUrl":"https://example.com/demo","slidesUrl":"https://example.com/slides"}')
step "create talk" "$R" 201
TALK=$(jqf "$(body "$R")" .id)

say "== 2. Events & CFPs =="
R=$(post /events '{"name":"PlatformCon 2027","description":"Platform engineering conference","websiteUrl":"https://platformcon.com","city":"London","country":"UK","eventType":"conference","topics":["platform engineering","kubernetes"],"status":"tracking","startsAt":"2027-06-01T09:00:00Z","endsAt":"2027-06-02T18:00:00Z"}')
step "create event" "$R" 201
EVENT=$(jqf "$(body "$R")" .id)

R=$(post /cfps "{\"eventId\":\"$EVENT\",\"name\":\"PlatformCon 2027 CFP\",\"submissionUrl\":\"https://platformcon.com/cfp\",\"opensAt\":\"2026-10-01T00:00:00Z\",\"closesAt\":\"2026-12-01T00:00:00Z\",\"tracks\":[\"platform engineering\"],\"requirements\":\"30 min talk\",\"status\":\"open\"}")
step "create CFP" "$R" 201
CFP=$(jqf "$(body "$R")" .id)

say "== 3. Opportunity intelligence =="
R=$(get "/opportunities/cfps")
step "rank CFP/talk fit" "$R" 200
N=$(jqf "$(body "$R")" 'length'); [ "${N:-0}" -ge 1 ] && ok "CFP opportunity produced ($N)" || bad "CFP opportunity produced" "got $N"
SCORE=$(jqf "$(body "$R")" '.[0].score')
[ -n "$SCORE" ] && [ "$SCORE" != "null" ] && ok "fit score present ($SCORE)" || bad "fit score present" "missing"
REASONS=$(jqf "$(body "$R")" '.[0].reasons | length')
[ "${REASONS:-0}" -ge 1 ] && ok "score is explainable ($REASONS reasons)" || bad "score is explainable" "no reasons"

say "== 4. Submission pipeline =="
R=$(post /submissions "{\"cfpId\":\"$CFP\",\"talkId\":\"$TALK\",\"notes\":\"drafting\",\"status\":\"draft\"}")
step "create submission" "$R" 201
SUB=$(jqf "$(body "$R")" .id)
R=$(ccurl -w '\n%{http_code}' -X PATCH -H 'Content-Type: application/json' -d '{"status":"submitted"}' "$API/submissions/$SUB/status")
step "advance submission to submitted" "$R" 200

say "== 5. Community & relationship =="
R=$(post /communities '{"name":"Cloud Native Colombo","platform":"meetup","websiteUrl":"https://meetup.com/cnc","city":"Colombo","country":"Sri Lanka","topics":["kubernetes","platform engineering"],"memberCount":1200,"activityScore":78,"speakingFitScore":80,"status":"active","nextEventAt":"2026-09-20T18:00:00Z"}')
step "create community" "$R" 201
COMM=$(jqf "$(body "$R")" .id)

R=$(post /relationships "{\"communityId\":\"$COMM\",\"stage\":\"warm\",\"strength\":55,\"notes\":\"met organizer at KubeCon\",\"nextFollowUpAt\":\"2026-09-15T09:00:00Z\"}")
step "create relationship with follow-up" "$R" 201

say "== 6. Speaking opportunities (redesigned candidate selection) =="
R=$(get "/opportunities/speaking")
step "rank speaking opportunities" "$R" 200
N=$(jqf "$(body "$R")" 'length'); [ "${N:-0}" -ge 1 ] && ok "speaking opportunity produced ($N)" || bad "speaking opportunity produced" "got $N"
TF=$(jqf "$(body "$R")" '.[0].breakdown.topicFit')
[ "${TF:-0}" -gt 0 ] && ok "topic fit computed ($TF)" || bad "topic fit computed" "got $TF"

say "== 7. Unified calendar =="
R=$(get "/calendar?from=2026-09-01T00:00:00Z&to=2027-08-01T00:00:00Z")
step "load calendar" "$R" 200
KINDS=$(jqf "$(body "$R")" '[.items[].kind] | unique | join(",")')
say "        kinds present: ${KINDS:-none}"
echo "$KINDS" | grep -q cfp   && ok "CFP deadline on calendar"   || bad "CFP deadline on calendar" "kinds=$KINDS"
echo "$KINDS" | grep -q event && ok "event on calendar"          || bad "event on calendar" "kinds=$KINDS"
echo "$KINDS" | grep -q relationship_follow_up && ok "follow-up on calendar" || bad "follow-up on calendar" "kinds=$KINDS"

say "== 8. Signal Radar =="
i=1
while [ $i -le 4 ]; do
  R=$(post /signals "{\"provider\":\"manual\",\"title\":\"Helm upgrade keeps failing on our cluster ($i)\",\"body\":\"Every helm upgrade breaks and the docs are unclear about rollback. This is painful and manual.\",\"authorHandle\":\"dev$i\",\"topics\":[\"kubernetes\"],\"relevanceScore\":80,\"engagementScore\":20,\"status\":\"new\"}")
  c=$(code "$R"); [ "$c" = "201" ] || bad "create signal $i" "HTTP $c: $(body "$R" | head -c 150)"
  i=$((i+1))
done
[ "$FAIL" -eq 0 ] && ok "created 4 developer signals" || true
R=$(get /signals); step "list signals" "$R" 200

say "== 9. Pain-point clustering =="
R=$(ccurl -w '\n%{http_code}' -X POST "$API/pain-points/rebuild")
step "rebuild pain-point clusters" "$R" 200
R=$(get /pain-points); step "list pain points" "$R" 200
N=$(jqf "$(body "$R")" 'length'); [ "${N:-0}" -ge 1 ] && ok "pain point clustered ($N)" || bad "pain point clustered" "got $N"
PP=$(jqf "$(body "$R")" '.[0].id')
EV=$(jqf "$(body "$R")" '.[0].evidenceCount')
[ "${EV:-0}" -ge 2 ] && ok "cluster is evidence-backed ($EV signals)" || bad "cluster is evidence-backed" "evidence=$EV"

say "== 10. Pain point -> Action Queue -> Content =="
R=$(post "/pain-points/$PP/work-items" '{"kind":"content_brief","owner":"savi"}')
step "convert pain point to work item" "$R" 201
WORK=$(jqf "$(body "$R")" .id)
R=$(post "/work-items/$WORK/content-assets" '{"channel":"blog","format":"article","audience":"platform engineers"}')
step "create content asset from work item" "$R" 201
CONTENT=$(jqf "$(body "$R")" .id)
BRIEF=$(jqf "$(body "$R")" '.brief | tostring | length')
[ "${BRIEF:-0}" -gt 20 ] && ok "content brief carries evidence" || bad "content brief carries evidence" "brief len=$BRIEF"

say "== 11. Pain point -> Developer Feedback =="
R=$(post "/pain-points/$PP/feedback" '{"component":"helm","owner":"savi","githubRepository":"savisaluwadana/DevRelOS"}')
step "convert pain point to feedback" "$R" 201
FB=$(jqf "$(body "$R")" .id)

say "== 12. Campaigns & attribution =="
R=$(post /campaigns '{"name":"Q4 Platform Engineering Push","objective":"awareness","status":"active","startsAt":"2026-09-01T00:00:00Z","endsAt":"2026-12-31T00:00:00Z"}')
step "create campaign" "$R" 201
CAMP=$(jqf "$(body "$R")" .id)
for pair in "content_asset:$CONTENT" "work_item:$WORK" "submission:$SUB" "feedback:$FB" "community:$COMM" "talk:$TALK"; do
  t=$(echo "$pair" | cut -d: -f1); id=$(echo "$pair" | cut -d: -f2)
  R=$(post "/campaigns/$CAMP/items" "{\"entityType\":\"$t\",\"entityId\":\"$id\",\"channel\":\"blog\"}")
  c=$(code "$R"); [ "$c" = "201" ] || [ "$c" = "200" ] && ok "attribute $t" || bad "attribute $t" "HTTP $c: $(body "$R" | head -c 150)"
done
R=$(get "/campaigns/$CAMP/report"); step "campaign report" "$R" 200
OUT=$(jqf "$(body "$R")" '.outcomeScore')
[ -n "$OUT" ] && [ "$OUT" != "null" ] && ok "outcome score derived ($OUT)" || bad "outcome score derived" "missing"

say "== 13. Report carries its attributed items =="
# Regression guard: the report declared an "items" field and never filled it,
# so API and MCP consumers saw an empty list while linkedByType showed counts.
R=$(get "/campaigns/$CAMP/report")
RI=$(jqf "$(body "$R")" '.items | length')
LT=$(jqf "$(body "$R")" '[.linkedByType[]] | add')
[ "${RI:-0}" -gt 0 ] && ok "report embeds attributed items ($RI)" || bad "report embeds attributed items" "items=$RI while linkedByType totals $LT"
[ "${RI:-0}" = "${LT:-x}" ] && ok "report items agree with linkedByType" || bad "report items agree with linkedByType" "items=$RI linkedByType=$LT"

say "== 14. Measure responds to real outcomes =="
BEFORE=$(jqf "$(get "/campaigns/$CAMP/report" | sed '$d')" '.outcomeScore')
ccurl -o /dev/null -X PATCH -H 'Content-Type: application/json' -d '{"status":"done"}' "$API/work-items/$WORK/status"
ccurl -o /dev/null -X PATCH -H 'Content-Type: application/json' -d '{"status":"accepted"}' "$API/submissions/$SUB/status"
AFTER=$(jqf "$(get "/campaigns/$CAMP/report" | sed '$d')" '.outcomeScore')
if [ "${AFTER:-0}" -gt "${BEFORE:-0}" ]; then
  ok "outcome score tracks real outcomes ($BEFORE -> $AFTER)"
else
  bad "outcome score tracks real outcomes" "stayed at $BEFORE after work completed and a talk was accepted"
fi

say "== 15. Command Center =="
R=$(get /dashboard); step "dashboard" "$R" 200
# Regression guard: this panel filtered on cfps.fit_score, a column nothing
# computes, so it was permanently empty however well a CFP scored.
NEWEV=$(post /events '{"name":"KubeCon EU 2027","eventType":"conference","topics":["kubernetes","platform engineering"],"status":"tracking","startsAt":"2027-03-15T09:00:00Z"}')
NEWEVID=$(jqf "$(body "$NEWEV")" .id)
post "/cfps" "{\"eventId\":\"$NEWEVID\",\"name\":\"KubeCon EU 2027 CFP\",\"submissionUrl\":\"https://kubecon.io/cfp\",\"closesAt\":\"2026-11-15T00:00:00Z\",\"tracks\":[\"platform engineering\",\"kubernetes\"],\"status\":\"open\"}" >/dev/null
HF=$(jqf "$(get /dashboard | sed '$d')" '.highFitCfps | length')
[ "${HF:-0}" -ge 1 ] && ok "high-fit CFP panel populates ($HF)" || bad "high-fit CFP panel populates" "empty despite a high-scoring open CFP"
HFS=$(jqf "$(get /dashboard | sed '$d')" '.highFitCfps[0].fitScore')
[ "${HFS:-0}" -ge 70 ] && ok "panel reports the computed fit score ($HFS)" || bad "panel reports the computed fit score" "got $HFS"

say "== 16. Relationship Radar =="
R=$(get /relationships/radar); step "relationship radar" "$R" 200

printf '\n===== %d passed, %d failed =====\n' "$PASS" "$FAIL"
[ "$FAIL" -eq 0 ]

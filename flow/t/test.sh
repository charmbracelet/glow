#!/bin/bash

set -euo pipefail

TOP=$(git rev-parse --show-toplevel) \
    && cd "$TOP"

((${GLOW_TEST_LOG:-0})) \
    && exec 3>/dev/stdout \
    || exec 3>/dev/null

GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

total=0
passed=0
failed=0

if (($# < 1)); then
    set -- go flow/t/test_*.sh
fi

go build \
    || { echo -e "${RED}go build failed!${NC}"; exit 1; }

echo -e "🎯🎯🎯 THE 'YAAAY! GREEN SMILES' GLOW FLOW VALIDATION SUITE 🎯🎯🎯"

TIMEFMT=$'\n'"  ⏱️ flow/t/test.sh (%P CPU over %*Es)"
TIMEFORMAT=$'\n'"  ⏱️ flow/t/test.sh (%P CPU over %3lR)"
time for t in "$@"; do
    if [ "$t" == "go" ]; then
        ((total += 1))
        (
            echo -e "\n> go test ./flow/...\n"
            TIMEFMT="    ⏳ go test ./flow/... (%P CPU over %*Es)"
            TIMEFORMAT="    ⏳ go test ./flow/... (%P CPU over %3lR)"
            time go test -v ./flow/... | sed -u 's,^,    ,' >&3 \
            && echo -e "    ✅ flow/t/test.sh ${GREEN}go test ./flow/... EXIT(0)${NC}" \
            && exit 0 \
            || echo -e "    ❌ flow/t/test.sh ${RED}go test ./flow/... EXIT($?)${NC}"
            exit 1
        ) \
        && ((passed += 1)) \
        || ((failed += 1))
        ((failed <= ${GLOW_TEST_MAX_FAILED:-failed})) \
        && continue \
        || break
    fi
    if [ ! -x "$t" ]; then
        echo -e "${RED}Test script $t is not executable!${NC}"
        exit 1
    fi
    echo -e "\n> flow/t/test.sh $t ...\n"
    IFS=$' \n\t'
    both=( $(
        TIMEFMT="    ⏳ flow/t/test.sh $t (%P CPU over %*Es)"
        TIMEFORMAT="    ⏳ flow/t/test.sh $t (%P CPU over %3lR)"
        (
            time timeout -pvk 31s 30s "$t" 2>&1 \
            && echo -e "✅ flow/t/test.sh ${GREEN}$t EXIT(0)${NC}" \
            || echo -e "❌ flow/t/test.sh ${RED}$t EXIT($?)${NC}"
        ) \
        | tee >(sed 's,^,    ,' >&3) \
        | (echo ❌; echo ✅; grep -oE "❌|✅") \
        | LC_ALL=C sort \
        | uniq -c \
        | cut -wf2
    ) )
    pass=$((both[0] - 1))
    fail=$((both[1] - 1))
    if [ "$fail" -eq 0 ]; then
        echo -e "    🎯 flow/t/test.sh ${GREEN}$t ($pass/$((pass + fail)) tests passed)${NC}"
    else
        echo -e "    ⚠️ flow/t/test.sh ${RED}$t ($pass/$((pass + fail)) tests passed)${NC}"
    fi
    ((total += pass + fail))
    ((passed += pass))
    ((failed += fail))
    ((failed <= ${GLOW_TEST_MAX_FAILED:-failed})) || break
done

if ((passed == total)); then
    echo -e "\n🎉🎉🎉 ${GREEN}YAAAY! ALL $passed/$total TESTS PASSED!${NC} 🎉🎉🎉"
    sleep 1
    exit 0
fi

sleep 1
echo -e "\n⚠️⚠️⚠️ ${RED}NOPES! SRY $failed/$total TESTS FAILED!${NC} ⚠️⚠️⚠️"
exit 1

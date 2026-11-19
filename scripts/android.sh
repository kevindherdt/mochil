#!/bin/sh
# Android builds on this box have taken the whole server down: an R8 run sat at
# 80 MB/s and load 200 for half an hour while the library container and the
# gateway stopped answering. A build is a background job, not an outage, so
# every Gradle invocation goes through here:
#   - one worker, low scheduling priority (org.gradle.priority=low as well)
#   - idle I/O class, so reads and writes yield to the serving container
#   - a CPU quota of four cores out of twelve, so the majority stay free
#
# Usage: scripts/android.sh :app:testDirectDebugUnitTest
set -eu
root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$root/android"

# A release keystore, when one is present, signs local builds too. Without this the same build made
# here and made in CI would carry different keys, and an APK from either would refuse to install over
# the other. The file is gitignored; CI gets the same key from repository secrets.
if [ -f "$root/android/.signing/release.env" ]; then
  set -a
  . "$root/android/.signing/release.env"
  set +a
fi

# Robolectric unpacks its native runtime (fonts, ICU, ~160 MB) plus a temp dir per test into
# java.io.tmpdir, and Gradle/Kotlin daemons write scratch there too. On this box /tmp is a tmpfs, so all of
# it is host RAM: ~250 leftover dirs once sat there and took part in a global OOM. Route it to the data disk
# — `-Djava.io.tmpdir` for the test workers, since the JVM ignores TMPDIR.
test_tmp=""
if [ -d /mnt/data/tmp ] && [ -w /mnt/data/tmp ]; then
  test_tmp="-PtestTmpDir=/mnt/data/tmp"
  GRADLE_OPTS="${GRADLE_OPTS:-} -Djava.io.tmpdir=/mnt/data/tmp"
  export GRADLE_OPTS
  TMPDIR=/mnt/data/tmp
  export TMPDIR
  find /mnt/data/tmp -maxdepth 1 -name 'robolectric-*' -mtime +1 -exec rm -rf {} + 2>/dev/null || true
fi

# `systemctl is-system-running` exits nonzero on a merely degraded host (any
# unrelated failed unit), which silently skipped every cap below. Test for the
# manager's socket instead: degraded is exactly the state this protection is for.
if command -v systemd-run >/dev/null 2>&1 && [ -S /run/systemd/system ]; then
  exec systemd-run --scope --quiet --collect \
    -p CPUQuota=400% -p IOWeight=10 -p MemoryMax=6G \
    -- nice -n 19 ionice -c 3 flock /tmp/gradle.lock ./gradlew --offline --no-daemon --max-workers=1 $test_tmp "$@"
fi
exec nice -n 19 ionice -c 3 flock /tmp/gradle.lock ./gradlew --offline --no-daemon --max-workers=1 $test_tmp "$@"

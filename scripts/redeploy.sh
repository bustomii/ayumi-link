#!/bin/sh
# Redeploy Ayumi Link safely, from the Mac terminal OR from a chat running inside the container.
#
#   sh scripts/redeploy.sh            # rebuild + restart (host Docker access if the socket is mounted)
#
# Why this script exists: running `docker compose up` directly inside the Ayumi Link container
#  1. kills the very process that is running it (the container is replaced mid-command), and
#  2. makes compose read variables from the *container's* environment, which wins over .env, so
#     mounts such as ~/.claude can silently resolve differently than on the Mac (sessions "vanish").
# Inside a container this script therefore hands the work to a short-lived helper container on the
# host daemon: clean environment, same project path, and it survives the app being replaced.
set -eu

PROJ="$(cd "$(dirname "$0")/.." && pwd)"
SOCK="${DOCKER_SOCK:-/var/run/docker.sock}"
FILES="-f docker-compose.yml"
[ -f "$PROJ/docker-compose.host-docker.yml" ] && FILES="$FILES -f docker-compose.host-docker.yml"

if [ ! -f /.dockerenv ]; then
  # On the host: plain compose, nothing special.
  cd "$PROJ"
  # shellcheck disable=SC2086
  exec docker compose $FILES up -d --build
fi

# ---- inside a container ----
command -v docker >/dev/null 2>&1 || {
  echo "ERROR: no docker CLI in this container. Deploy once from the Mac with: make up-docker" >&2
  exit 1
}
[ -S "$SOCK" ] || { echo "ERROR: Docker socket $SOCK is not mounted (use docker-compose.host-docker.yml)." >&2; exit 1; }

# The helper mounts the project at the same path, so the path must exist on the host 1:1
# (i.e. the folder containing the project is mounted with WORKSPACE_DIR=<host path> and the
# same WORKSPACE_MOUNT). Otherwise the host daemon would mount a folder that does not exist.
case "$PROJ" in
  /workspace|/workspace/*|/home/*|/app/*|/data/*)
    echo "ERROR: project is at $PROJ inside the container, which is not its path on the host." >&2
    echo "Mount your projects 1:1 (WORKSPACE_DIR=/Users/you/Projects, WORKSPACE_MOUNT=/Users/you/Projects) or deploy from the Mac." >&2
    exit 1
    ;;
esac

NAME="ayumi-redeploy"
docker rm -f "$NAME" >/dev/null 2>&1 || true

# Runs detached on the host daemon with a clean environment. After the app is up it checks that the
# Claude sessions folder is really mounted, and says so if it is empty.
SCRIPT='
set -eu
sleep 3
docker compose '"$FILES"' up -d --build
for i in 1 2 3 4 5 6 7 8 9 10; do
  state="$(docker inspect -f "{{.State.Health.Status}}" ayumi-link 2>/dev/null || echo none)"
  [ "$state" = healthy ] && break
  sleep 3
done
n="$(docker exec ayumi-link sh -c "ls /home/node/.claude/projects 2>/dev/null | wc -l" || echo 0)"
echo "ayumi-link: health=$state, claude project folders visible=$n"
if [ "$n" = 0 ]; then echo "WARNING: /home/node/.claude looks empty. Check CLAUDE_HOME_DIR in .env."; fi
'

docker run -d --rm --name "$NAME" \
  -v "$SOCK":/var/run/docker.sock \
  -v "$PROJ":"$PROJ" -w "$PROJ" \
  --entrypoint sh docker:27-cli -c "$SCRIPT" >/dev/null

echo "Redeploy started in the background (helper container: $NAME)."
echo "This chat/terminal will disconnect when the app restarts; reload the page in ~1-2 minutes."
echo "Progress:  docker logs -f $NAME   (run on the Mac)"

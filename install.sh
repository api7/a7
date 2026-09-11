#!/bin/sh
# Install the API7 Enterprise Edition (a7) AI agent skill into your AI coding agent.
#
# The a7 skill is maintained in https://github.com/api7/agent-skills and teaches
# an agent (Claude Code, Cursor, Copilot, Windsurf, OpenCode, ...) how to
# configure a live API7 Enterprise Edition gateway through the a7 CLI. The recommended way to install it
# is the skills CLI, which needs Node.js:
#   npx skills add api7/agent-skills --skill a7
#
# This script is the no-Node fallback: it downloads the api7/agent-skills
# tarball and copies skills/a7 into your agent's skills directory as "a7".
#
# Quick start (installs into ~/.claude/skills/a7 for Claude Code):
#   curl -fsSL https://raw.githubusercontent.com/api7/a7/master/install.sh | sh
#
# Install somewhere else (e.g. a project-local Cursor rules dir):
#   curl -fsSL https://raw.githubusercontent.com/api7/a7/master/install.sh | sh -s -- --dir ./.cursor/rules
#   SKILLS_DIR=~/.config/opencode/skills sh -c "$(curl -fsSL https://raw.githubusercontent.com/api7/a7/master/install.sh)"
set -eu

REPO="api7/agent-skills"
BRANCH="main"
SKILL="a7"
LABEL="API7 Enterprise Edition"

# Target directory. Default: Claude Code personal skills. Override with
# SKILLS_DIR=... or --dir <path>.
SKILLS_DIR="${SKILLS_DIR:-$HOME/.claude/skills}"

while [ $# -gt 0 ]; do
  case "$1" in
    --dir) SKILLS_DIR="${2:?--dir needs a path}"; shift 2 ;;
    --dir=*)
      SKILLS_DIR="${1#*=}"
      [ -n "$SKILLS_DIR" ] || { echo "install.sh: --dir needs a path" >&2; exit 1; }
      shift
      ;;
    -h | --help)
      echo "Usage: install.sh [--dir <path>]   (default: \$HOME/.claude/skills)"
      echo "Installs the ${SKILL} skill from ${REPO} into <path>/${SKILL}."
      exit 0
      ;;
    *) echo "install.sh: unknown option '$1'" >&2; exit 1 ;;
  esac
done

if command -v curl >/dev/null 2>&1; then
  fetch() { curl -fsSL "$1"; }
elif command -v wget >/dev/null 2>&1; then
  fetch() { wget -qO- "$1"; }
else
  echo "install.sh: need curl or wget on your PATH." >&2
  exit 1
fi

echo "Installing the ${LABEL} agent skill (${SKILL}) into ${SKILLS_DIR}/${SKILL} ..."

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT INT TERM

# Download the api7/agent-skills tarball (no git required) and extract just
# the a7 skill.
fetch "https://codeload.github.com/${REPO}/tar.gz/refs/heads/${BRANCH}" >"$TMP/repo.tgz" ||
  { echo "install.sh: download failed." >&2; exit 1; }
tar -xzf "$TMP/repo.tgz" -C "$TMP"

SRC="$(find "$TMP" -maxdepth 3 -type d -path "*/skills/${SKILL}" | head -n 1)"
if [ -z "$SRC" ] || [ ! -f "$SRC/SKILL.md" ]; then
  echo "install.sh: could not find skills/${SKILL}/SKILL.md in the download." >&2
  exit 1
fi

mkdir -p "$SKILLS_DIR"
rm -rf "${SKILLS_DIR:?}/${SKILL}"
cp -R "$SRC" "${SKILLS_DIR}/${SKILL}"

echo "Installed the ${SKILL} skill to ${SKILLS_DIR}/${SKILL}"
echo
echo "Next: ask your AI coding agent to configure ${LABEL} in plain language, e.g."
echo "  \"add key-auth to my /orders route and rate-limit it to 100 requests per minute\""
echo
echo "Browse the skill: https://skills.sh/api7/agent-skills/${SKILL}"

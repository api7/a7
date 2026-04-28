#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SKILLS_DIR="${ROOT_DIR}/skills"

if [[ ! -d "${SKILLS_DIR}" ]]; then
  echo "skills directory not found: ${SKILLS_DIR}" >&2
  exit 1
fi

status=0
seen_names_file="$(mktemp)"
trap 'rm -f "${seen_names_file}"' EXIT

for skill_dir in "${SKILLS_DIR}"/*; do
  [[ -d "${skill_dir}" ]] || continue

  skill_name="$(basename "${skill_dir}")"
  skill_file="${skill_dir}/SKILL.md"

  if [[ ! -f "${skill_file}" ]]; then
    echo "${skill_name}: missing SKILL.md" >&2
    status=1
    continue
  fi

  if [[ "$(sed -n '1p' "${skill_file}")" != "---" ]]; then
    echo "${skill_name}: SKILL.md must start with YAML frontmatter delimiter" >&2
    status=1
    continue
  fi

  end_line="$(awk 'NR > 1 && $0 == "---" { print NR; exit }' "${skill_file}")"
  if [[ -z "${end_line}" ]]; then
    echo "${skill_name}: SKILL.md missing closing YAML frontmatter delimiter" >&2
    status=1
    continue
  fi

  frontmatter="$(sed -n "2,$((end_line - 1))p" "${skill_file}")"
  name="$(printf '%s\n' "${frontmatter}" | awk -F': *' '$1 == "name" { print $2; exit }' | tr -d '"'"'"'')"

  if [[ -z "${name}" ]]; then
    echo "${skill_name}: missing required frontmatter field: name" >&2
    status=1
  elif [[ "${name}" != "${skill_name}" ]]; then
    echo "${skill_name}: frontmatter name ${name} must match directory name" >&2
    status=1
  elif [[ ! "${name}" =~ ^[a-z0-9]+(-[a-z0-9]+)*$ ]]; then
    echo "${skill_name}: name must be kebab-case" >&2
    status=1
  fi

  if [[ -n "${name}" ]]; then
    if grep -Fxq "${name}" "${seen_names_file}"; then
      echo "${skill_name}: duplicate skill name ${name}" >&2
      status=1
    fi
    printf '%s\n' "${name}" >>"${seen_names_file}"
  fi

  description="$(printf '%s\n' "${frontmatter}" | awk '
    function ltrim(s) { sub(/^[[:space:]]+/, "", s); return s }
    function has_text(s) {
      s = ltrim(s)
      return s !~ /^[>|]-?$/ && s ~ /[^[:space:]]/
    }
    /^[A-Za-z0-9_-]+:/ {
      if (in_description && $0 !~ /^description:/) {
        exit
      }
    }
    /^description:[[:space:]]*/ {
      in_description = 1
      sub(/^description:[[:space:]]*/, "")
      if (has_text($0)) {
        print
      }
      next
    }
    in_description {
      if ($0 ~ /^[[:space:]]+/) {
        if (has_text($0)) {
          print
        }
        next
      }
      exit
    }
  ')"
  if [[ -z "${description}" ]]; then
    echo "${skill_name}: missing required frontmatter field: description" >&2
    status=1
  fi

  if [[ "$(sed -n "$((end_line + 1))p" "${skill_file}")" == "" ]]; then
    # Empty separator line is fine, but the body still needs content.
    body_start=$((end_line + 2))
  else
    body_start=$((end_line + 1))
  fi
  if ! tail -n +"${body_start}" "${skill_file}" | grep -q '[^[:space:]]'; then
    echo "${skill_name}: SKILL.md body must not be empty" >&2
    status=1
  fi
done

if [[ ${status} -eq 0 ]]; then
  echo "Validated $(wc -l <"${seen_names_file}" | tr -d ' ') skills"
fi

exit "${status}"

#!/usr/bin/env bash
# Безопасное обновление статистики профиля.
#
# Токен вводится вручную, не попадает ни в аргументы командной строки (значит не
# виден в `ps` у других процессов), ни в историю bash, ни на диск, ни в GitHub.
# В репозиторий уезжают только производные числа.
#
#   ./scripts/refresh.sh            # спросит токен, обновит SVG, закоммитит
#   ./scripts/refresh.sh --dry-run  # обновит SVG, без коммита и push

set -euo pipefail

cd "$(dirname "$0")/.."

LOGIN="${GH_USER:-ViktorVanyuta}"
DRY_RUN=0
[[ "${1:-}" == "--dry-run" ]] && DRY_RUN=1

command -v go >/dev/null || { echo "не найден go в PATH" >&2; exit 1; }

echo "Введи Personal access token. Ввод не отображается."
echo "Нужен fine-grained токен либо classic со scope 'repo'."
echo "Токен нигде не сохраняется и не уходит на GitHub."
echo
read -rsp "Token: " GITHUB_TOKEN
echo
export GITHUB_TOKEN

cleanup() { unset GITHUB_TOKEN; }
trap cleanup EXIT INT TERM

if [[ -z "${GITHUB_TOKEN}" ]]; then
  echo "Токен пустой, выхожу." >&2
  exit 1
fi

go run ./cmd/statsgen stats --user "$LOGIN" --out assets

if [[ $DRY_RUN -eq 1 ]]; then
  echo "dry-run: коммит и push пропущены"
  exit 0
fi

git add assets

# Страховка: не дать закоммитить токен, если он где-то утёк в рабочее дерево.
# Проверка идёт после git add, иначе она не видела бы сами SVG.
if ! go run ./cmd/statsgen check-leaks --staged-only; then
  git reset -q -- assets
  echo "Найдены строки, похожие на токены — коммит отменён." >&2
  exit 1
fi

if git diff --cached --quiet -- assets; then
  echo "Статистика не изменилась"
  exit 0
fi

git commit -q -m "chore(stats): обновить статистику профиля"
git push
echo "Готово. Токен не сохранён."

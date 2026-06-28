#!/usr/bin/env sh
# harness uninstaller — removes the globally installed binary.
#
# Usage:
#   ./uninstall.sh
#   INSTALL_DIR=~/bin ./uninstall.sh
set -eu

APP_NAME="harness"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
TARGET="$INSTALL_DIR/$APP_NAME"

if [ ! -e "$TARGET" ]; then
	echo "$APP_NAME is not installed at $TARGET. Nothing to do."
	exit 0
fi

echo "Removing $TARGET ..."
if [ -w "$INSTALL_DIR" ]; then
	rm -f "$TARGET"
else
	sudo rm -f "$TARGET"
fi
echo "Removed. (Your shared library at ~/.harness was left untouched.)"

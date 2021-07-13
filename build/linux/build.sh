#!/bin/sh

if [ $# -ne 1 ]; then
  echo "Usage: $0 <Version>"
  exit 1
fi

buildnum=$(date '+%Y%m%d%H%M%S')
~/go/bin/fyne package -appBuild $buildnum -appID com.github.hannesbraun.garrett -appVersion $1 -name Garrett -os linux -sourceDir ../../ -release -icon ../../Icon.png

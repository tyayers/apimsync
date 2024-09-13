if [ "${APIMSYNC_VERSION}" = "" ] ; then
  APIMSYNC_VERSION="$(curl -si  https://api.github.com/repos/tyayers/apimsync/releases/latest | grep tag_name | sed -E 's/.*"([^"]+)".*/\1/')"
fi

echo "Downloading apimsync version: $APIMSYNC_VERSION"

sudo curl -o /usr/bin/apimsync -fsLO "https://github.com/tyayers/apimsync/releases/download/$APIMSYNC_VERSION/apimsync"
sudo chmod +x /usr/bin/apimsync
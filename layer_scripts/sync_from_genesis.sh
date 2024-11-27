export FILE_DESTINATION_ROOT_DIR_NAME=layer_binaries

mkdir -p $HOME/$FILE_DESTINATION_ROOT_DIR_NAME

export URL=https://github.com/tellor-io/layer/releases/download/v2.0.0-alpha1/layer_Linux_x86_64.tar.gz
export FILENAME_FOR_OS=layer_Linux_x86_64.tar.gz
export DAEMON_NAME=layerd
export DAEMON_HOME=$HOME/layer/layerd

echo "You must have cosmovisor installed and setup for this script to work...."
sleep 2

for VERSION in v2.0.0-alpha1; do

    echo "Downloading $FILE_NAME..."
    wget -q --show-progress "https://github.com/tellor-io/layer/releases/download/$VERSION/$FILENAME_FOR_OS" -O "$HOME/$FILE_DESTINATION_ROOT_DIR_NAME/$FILENAME_FOR_OS"
    if [ $? -ne 0 ]; then
        echo "Error: Download failed!"
        exit 1
    fi

    # Step 3: Decompress the File
    echo "Decompressing $VERSION binary..."
    tar -xzvf "$HOME/$FILE_DESTINATION_ROOT_DIR_NAME/$FILENAME_FOR_OS" -C "$HOME/$FILE_DESTINATION_ROOT_DIR_NAME/layerd_$VERSION"
    if [ $? -ne 0 ]; then
        echo "Error: Decompression failed!"
        exit 1
    fi

    rm -rf $HOME/$FILE_DESTINATION_ROOT_DIR_NAME/$FILENAME_FOR_OS

    echo "Adding $VERSION upgrade to cosmovisor..."
    $HOME/cosmovisor add-upgrade $VERSION $HOME/$FILE_DESTINATION_ROOT_DIR_NAME/layerd_$VERSION

done

echo "All upgrades are setup in cosmovisor and you are ready to start syncing from genesis"
echo "Start your cosmovisor service or call ./cosmovisor start with your flags from layer start command"



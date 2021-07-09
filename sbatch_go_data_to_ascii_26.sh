#!/bin/bash -x
#SBATCH --nodes=1
#SBATCH --ntasks=1
#SBATCH --cpus-per-task=80
#SBATCH --partition=compute
#SBATCH --job-name=go_ascii
#SBATCH --time=01:00:00


# download go image
IMAGE_DIR_GO=~/singularity/other
SINGULARITY_GO_IMAGE=golang_1.14.4.sif
IMAGE_GO_PATH=${IMAGE_DIR_GO}/${SINGULARITY_GO_IMAGE}
mkdir -p $IMAGE_DIR_GO
if [ ! -e ${IMAGE_GO_PATH} ] ; then
echo "File '${IMAGE_GO_PATH}' not found"
cd $IMAGE_DIR_GO
singularity pull docker://golang:1.14.4
cd ~
fi

IMAGE_DIR_PYTHON=~/singularity/python
SINGULARITY_PYTHON_IMAGE=python3.7_2.0.sif
IMAGE_PYTHON_PATH=${IMAGE_DIR_PYTHON}/${SINGULARITY_PYTHON_IMAGE}
mkdir -p $IMAGE_DIR_PYTHON
if [ ! -e ${IMAGE_PYTHON_PATH} ] ; then
echo "File '${IMAGE_PYTHON_PATH}' not found"
cd $IMAGE_DIR_PYTHON
singularity pull docker://zalfrpm/python3.7:2.0
cd ~
fi

cd ~/go/src/github.com/cropMapsEU
singularity run ~/singularity/other/golang_1.14.4.sif go get github.com/cheggaaa/pb && \
    go get gonum.org/v1/gonum/stat && \
    go build -v -o DataToAscii

mkdir -p rpc_26
./DataToAscii -path Cluster -crop maize -source /beegfs/rpm/projects/monica/out/sschulz_2346_2021-08-July_162754 -project /beegfs/rpm/projects/monica/project/soybeanEU -out ./rpc_26
./DataToAscii -path Cluster -crop wheat -source /beegfs/rpm/projects/monica/out/sschulz_2346_2021-08-July_162754 -project /beegfs/rpm/projects/monica/project/soybeanEU -out ./rpc_26


FOLDER=$( pwd )
IMG=~/singularity/python/python3.7_2.0.sif
singularity run -B $FOLDER/rpc_26/asciigrid:/source,$FOLDER/rpc_26:/out $IMG python create_image_from_ascii.py path=cluster
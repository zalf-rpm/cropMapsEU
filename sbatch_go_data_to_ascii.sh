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
cd ~/go/src/github.com/cropMapsEU
singularity run ~/singularity/other/golang_1.14.4.sif go get github.com/cheggaaa/pb && \
    go get gonum.org/v1/gonum/stat && \
    go build -v -o DataToAscii

./DataToAscii -path Cluster -source /beegfs/rpm/projects/monica/out/sschulz_2274_2021-28-May_131548 -project /beegfs/rpm/projects/monica/project/soybeanEU -climate /beegfs/common/data/climate/macsur_european_climate_scenarios_v3/testing/corrected -out .



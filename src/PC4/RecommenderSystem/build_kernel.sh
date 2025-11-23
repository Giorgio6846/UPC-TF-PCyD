#!/bin/bash

export PATH=/usr/local/cuda/bin:$PATH
export LD_LIBRARY_PATH=/usr/local/cuda/lib64:$LD_LIBRARY_PATH

nvcc -O3 --compiler-options '-fPIC' \
  -shared cosine_sim.cu -o libcosine_sparse.so

echo "Build complete."
// cosine_sparse_gpu.cu
#include <cuda_runtime.h>
#include <thrust/device_vector.h>
#include <thrust/host_vector.h>
#include <thrust/set_operations.h>
#include <thrust/iterator/zip_iterator.h>
#include <thrust/transform_reduce.h>
#include <thrust/functional.h>
#include <cmath>

struct Square {
    __host__ __device__
    double operator()(const double& x) const { return x * x; }
};

struct MulPair {
    __host__ __device__
    double operator()(const thrust::tuple<double,double>& t) const {
        return thrust::get<0>(t) * thrust::get<1>(t);
    }
};

extern "C" double CosineSparseF64(
    const int* idxA_h, const double* valA_h, int nnzA,
    const int* idxB_h, const double* valB_h, int nnzB
) {
    if (nnzA == 0 || nnzB == 0) return 0.0;

    // --- Copy inputs to device ---
    thrust::device_vector<int> idxA_d(idxA_h, idxA_h + nnzA);
    thrust::device_vector<double> valA_d(valA_h, valA_h + nnzA);

    thrust::device_vector<int> idxB_d(idxB_h, idxB_h + nnzB);
    thrust::device_vector<double> valB_d(valB_h, valB_h + nnzB);

    // --- Norms: sqrt(sum(val^2)) ---
    double na2 = thrust::transform_reduce(
        valA_d.begin(), valA_d.end(),
        Square(),
        0.0,
        thrust::plus<double>()
    );

    double nb2 = thrust::transform_reduce(
        valB_d.begin(), valB_d.end(),
        Square(),
        0.0,
        thrust::plus<double>()
    );

    double na = std::sqrt(na2);
    double nb = std::sqrt(nb2);
    double denom = na * nb;
    if (denom == 0.0) return 0.0;

    // --- Intersection-by-key to get matching indices ---
    thrust::device_vector<int> idxOut_d(std::min(nnzA, nnzB));
    thrust::device_vector<double> valAOut_d(std::min(nnzA, nnzB));
    thrust::device_vector<double> valBOut_d(std::min(nnzA, nnzB));

    auto endIt = thrust::set_intersection_by_key(
        idxA_d.begin(), idxA_d.end(),
        idxB_d.begin(), idxB_d.end(),
        valA_d.begin(),
        valB_d.begin(),
        idxOut_d.begin(),
        valAOut_d.begin(),
        valBOut_d.begin()
    );

    int common = endIt.first - idxOut_d.begin();
    if (common == 0) return 0.0;

    // --- Dot over common entries: sum(valA_common * valB_common) ---
    auto zipped = thrust::make_zip_iterator(
        thrust::make_tuple(valAOut_d.begin(), valBOut_d.begin())
    );

    double dot = thrust::transform_reduce(
        zipped, zipped + common,
        MulPair(),
        0.0,
        thrust::plus<double>()
    );

    return dot / denom;
}

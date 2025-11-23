//package recommendersystem
//
//import (
//	"C"
//	"pc4/tools"
//	"sort"
//	"unsafe"
//)
//
//func flattenSparse(v map[int]float64) (idx []int32, val []float64) {
//	idx = make([]int32, 0, len(v))
//	for k := range v {
//		idx = append(idx, int32(k))
//	}
//	sort.Slice(idx, func(i, j int) bool { return idx[i] < idx[j] })
//
//	val = make([]float64, len(idx))
//	for i, k := range idx {
//		val[i] = v[int(k)]
//	}
//	return
//}
//
//func CosineSimilaritySparseCUDA(a, b tools.UserVector) float64 {
//	idxA, valA := flattenSparse(a)
//	idxB, valB := flattenSparse(b)
//
//	if len(idxA) == 0 || len(idxB) == 0 {
//		return 0
//	}
//
//	res := C.CosineSparseF64(
//		(*C.int)(unsafe.Pointer(&idxA[0])),
//		(*C.double)(unsafe.Pointer(&valA[0])),
//		C.int(len(idxA)),
//		(*C.int)(unsafe.Pointer(&idxB[0])),
//		(*C.double)(unsafe.Pointer(&valB[0])),
//		C.int(len(idxB)),
//	)
//	return float64(res)
//}

package recommendersystem

import "fmt"

func Test() {
	fmt.Printf("")
}

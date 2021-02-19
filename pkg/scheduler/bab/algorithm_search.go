package bab

import (
	"math"
)

func (node *NodeInfo) Search(pods fiterPods) int {
	core := node.Allocate.Core
	mem := node.Allocate.Mem
	high := node.Index
	for _, pod := range pods[high:] {
		if pod.Weight.Core > node.Allocate.Core || pod.Weight.Mem > node.Allocate.Mem {
			break
		}
		core = core - pod.Weight.Core
		mem = mem - pod.Weight.Mem
		high += 1
	}
	if high == len(pods) {
		return -1
	}

	target := float64(core*mem) / float64(node.Capicity.Core*node.Capicity.Mem)
	low := len(pods) - 1

	for (low - high) != 1 {
		mid := int(math.Ceil(float64((high + low) / 2.0)))
		pod := pods[mid]
		if pod.Usage > target {
			high = mid
		} else {
			low = mid
		}
	}

	for _, pod := range pods[high:] {
		if pod.Weight.Core <= node.Allocate.Core && pod.Weight.Mem <= node.Allocate.Mem {
			return high
		}
		high += 1
	}
	return -1
}

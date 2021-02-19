package bab

import (
	"container/heap"
	"k8s.io/api/core/v1"
	"sort"
)

func ScheduleNode(pods []*v1.Pod, host *v1.Node) []int {
	//node resources infomation
	resourceCapiciyCpu := host.Status.Capacity[v1.ResourceCPU]
	resouceCapicityMem := host.Status.Capacity[v1.ResourceMemory]
	maxCore := resourceCapiciyCpu.MilliValue()
	maxMem := resouceCapicityMem.Value()
	capacity := Weight{
		Core: maxCore,
		Mem:  maxMem,
	}
	resouceAllocateCpu := host.Status.Allocatable[v1.ResourceCPU]
	resouceAllocateMem := host.Status.Allocatable[v1.ResourceMemory]
	allocateCore := resouceAllocateCpu.MilliValue()
	allocateMem := resouceAllocateMem.Value()
	allocate := Weight{
		Core: allocateCore,
		Mem:  allocateMem,
	}
	CapicityCoreAndMen := float64(maxCore * maxMem)
	//translate v1.Pod to PodInfo
	shedulingPods := NewFilterPods(pods, CapicityCoreAndMen)
	//preset of branch and bound
	nodes := make(PriorityQueue, 1)
	root := NewNodeInfo(capacity)
	root.Allocate = allocate
	root.SetIndexs(shedulingPods)
	root.Profit = 1 - (float64(allocateCore*allocateMem) / float64(maxCore*maxMem))
	root.SetBound(shedulingPods)
	bestNode := root
	nodes[0] = root
	heap.Init(&nodes)
	//branch and bound start
	for len(nodes) != 0 {
		v := heap.Pop(&nodes).(*NodeInfo)
		if v.Bound > bestNode.Profit {
			children := v.FindChildren(shedulingPods)
			for _, child := range children {
				if child.Profit > bestNode.Profit {
					bestNode = child
				}
				if child.Bound > bestNode.Bound {
					heap.Push(&nodes, child)
				}
			}
		}
	}
	takePath := bestNode.ShowChoices()
	result := make([]int, 1)
	for _, takePod := range takePath {
		result = append(result, takePod.Index)
	}
	sort.Sort(sort.IntSlice(result))
	return result
}

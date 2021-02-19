package bab

import (
	v1 "k8s.io/api/core/v1"
)

//"k8s.io/apimachinery/pkg/api/resource"

type PodInfo struct {
	//generated id(unique)
	Id string
	//index in []v1.Pod
	Index int
	//pod request
	Weight
	//node usage
	Usage float64
}

func NewPodInfo(pod *v1.Pod) *PodInfo {
	var RequestedCore, RequestedMen int64
	RequestedCore = 0
	RequestedMen = 0
	/*
		for _, container := range pod.Spec.InitContainers {
			RequestedCore += container.Resources.Requests[v1.ResourceCPU].MilliValue()
			RequestedMen += container.Resources.Requests[v1.ResourceMemory].Value()
		}
	*/
	for _, container := range pod.Spec.Containers {
		resourceCpu := container.Resources.Requests[v1.ResourceCPU]
		resouceMem := container.Resources.Requests[v1.ResourceMemory]
		RequestedCore += resourceCpu.MilliValue()
		RequestedMen += resouceMem.Value()
	}
	return &PodInfo{
		Id: pod.ObjectMeta.Name,
		Weight: Weight{
			Core: RequestedCore,
			Mem:  RequestedMen,
		},
	}
}

func (pod *PodInfo) ComputeUsage(CapicityCoreAndMen float64) {
	result := float64(pod.Weight.Core*pod.Weight.Mem) / CapicityCoreAndMen
	pod.Usage = result
}

func (pod *PodInfo) SetIndex(index int) {
	pod.Index = index
}

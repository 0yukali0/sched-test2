package bab

import (
	"sort"

	v1 "k8s.io/api/core/v1"
)

type fiterPods []*PodInfo

func (a fiterPods) Len() int {
	return len(a)
}
func (a fiterPods) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a fiterPods) Less(i, j int) bool {
	return a[i].Usage < a[j].Usage
}

func NewFilterPods(pods []*v1.Pod, CapicityCoreAndMen float64) fiterPods {
	podInfos := make(fiterPods, 1)
	for index, pod := range pods {
		mypod := NewPodInfo(pod)
		mypod.SetIndex(index)
		mypod.ComputeUsage(CapicityCoreAndMen)
		podInfos = append(podInfos, mypod)
	}
	sort.Sort(podInfos)
	return podInfos
}

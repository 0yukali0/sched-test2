package bab

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog"
	"sort"
)

type fiterPods []*PodInfo

func (a fiterPods) Len() int {
	return len(a)
}
func (a fiterPods) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}
func (a fiterPods) Less(i, j int) bool {
	return a[i].Usage > a[j].Usage
}

func NewFilterPods(pods []*v1.Pod, CapicityCoreAndMen float64) fiterPods {
	podInfos := make(fiterPods, 1)
	for index, pod := range pods {
		klog.Infof("filter pod %s ,%v", pod.Name, pod)
		mypod := NewPodInfo(pod)
		mypod.SetIndex(index)
		mypod.ComputeUsage(CapicityCoreAndMen)
		podInfos = append(podInfos, mypod)
		klog.Infof("my pod %s ,%v", pod.Name, mypod)
	}
	if len(podInfos) == 1 {
		return podInfos
	}
	sort.Sort(podInfos)
	return podInfos
}

package core

import (
	v1 "k8s.io/api/core/v1"
)

func (g *genericScheduler) PubToPredict(pod *v1.Pod, nodes []*v1.Node) (hosts []*v1.Node) {
	if err := podPassesBasicChecks(pod, g.pvcLister); err != nil {
		return hosts
	}

	if err := g.snapshot(); err != nil {
		return hosts
	}

	hosts, _, _ = g.findNodesThatFit(pod, nodes)
	return hosts
}

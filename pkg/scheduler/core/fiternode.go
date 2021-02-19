package core

import (
	v1 "k8s.io/api/core/v1"
)

func (g *genericScheduler) PubToPredict(pod *v1.Pod, nodes []*v1.Node) []*v1.Node {
	hosts, _, _ := g.findNodesThatFit(pod, nodes)
	return hosts
}

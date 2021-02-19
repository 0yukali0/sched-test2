package bab

type Weight struct {
	Core int64
	Mem  int64
}

type NodeInfo struct {
	Parent *NodeInfo
	//b&b use
	Profit   float64
	Allocate Weight
	Bound    float64
	Capicity Weight
	//is pod should be choose and pod index in queue
	Pod        *PodInfo
	Take       bool
	Index      int
	SearchJump int
	//heap index
	QIndex int
}

func NewNodeInfo(capicity Weight) *NodeInfo {
	return &NodeInfo{
		Parent:     nil,
		Profit:     0,
		Capicity:   capicity,
		Bound:      0,
		Take:       false,
		Pod:        nil,
		SearchJump: -1,
	}
}

func (node *NodeInfo) SetProfit(pods fiterPods) {
	//default nontake
	node.Profit = node.Parent.Profit
	if node.Take {
		//take pod and show resource usage
		node.Profit = 1 - (float64(node.Allocate.Core*node.Allocate.Mem) / float64(node.Capicity.Core*node.Capicity.Mem))
	}
}
func (node *NodeInfo) SetWeight(pods fiterPods) {
	//default nontake
	node.Allocate.Core = node.Parent.Allocate.Core
	node.Allocate.Mem = node.Parent.Allocate.Mem
	if node.Take {
		//take pod
		node.Allocate.Core = node.Allocate.Core - node.Parent.Pod.Weight.Core
		node.Allocate.Mem = node.Allocate.Mem - node.Parent.Pod.Weight.Mem
	}
}

func (node *NodeInfo) SetBound(pods fiterPods) {
	if node.Index == -1 {
		node.Bound = node.Profit
		return
	}

	core := node.Allocate.Core
	mem := node.Allocate.Mem
	index := node.Index
	//sequnce cost
	for _, pod := range pods[index:] {
		if pod.Weight.Core > node.Allocate.Core || pod.Weight.Mem > node.Allocate.Mem {
			break
		}
		core = core - pod.Weight.Core
		mem = mem - pod.Weight.Mem
	}

	//jump
	if node.SearchJump != -1 {
		core = core - pods[node.SearchJump].Weight.Core
		mem = mem - pods[node.SearchJump].Weight.Mem
	}
	node.Bound = 1 - float64((core*mem))/float64((node.Capicity.Core*node.Capicity.Mem))
}

func (node *NodeInfo) SetIndexs(pods fiterPods) {
	//index set
	if node.Parent == nil {
		//root
		node.Index = 0
	} else {
		//child
		node.Index = node.Parent.Index + 1
		if node.Index >= len(pods) {
			node.Index = -1
			node.SearchJump = -1
			return
		}

		if pods[node.Index].Weight.Core > node.Allocate.Core || pods[node.Index].Weight.Mem > node.Allocate.Mem {
			//out of sequence and no jump -> means no candicate
			if node.Parent.SearchJump == -1 {
				return
			}
			//out of sequence but has jump
			node.Index = node.Parent.SearchJump
		}
	}
	//find jumpindex
	node.SearchJump = node.Search(pods)
}

func (node *NodeInfo) SetPod(pods fiterPods) {
	if node.Index != -1 {
		node.Pod = pods[node.Index]
	} else {
		node.Pod = nil
	}
}

func (node *NodeInfo) FindChildren(pods fiterPods) []*NodeInfo {
	result := make([]*NodeInfo, 1)
	takePod := NewNodeInfo(node.Capicity)
	takePod.Parent = node
	takePod.Take = true
	takePod.SetWeight(pods)
	takePod.SetIndexs(pods)
	takePod.SetPod(pods)
	takePod.SetProfit(pods)
	takePod.SetBound(pods)

	dissPod := NewNodeInfo(node.Capicity)
	dissPod.Parent = node
	dissPod.Take = false
	dissPod.SetWeight(pods)
	dissPod.SetIndexs(pods)
	dissPod.SetPod(pods)
	dissPod.SetProfit(pods)
	dissPod.SetBound(pods)

	result = append(result, takePod)
	result = append(result, dissPod)
	return result
}

func (node *NodeInfo) ShowChoices() []*PodInfo {
	choices := make([]*PodInfo, 1)
	for choice := node.Parent; choice != nil; choice = choice.Parent {
		if choice.Take {
			choices = append(choices, choice.Parent.Pod)
		}
	}
	return choices
}

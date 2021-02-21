package scheduler

import (
	"errors"
	"fmt"

	v1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	"k8s.io/kubernetes/pkg/scheduler/bab"
	"k8s.io/kubernetes/pkg/scheduler/metrics"
)

type kubeNode struct {
	Index      int
	Node       *v1.Node
	Scheduling []int
}

type schedulingPod struct {
	Pod   *v1.Pod
	Index int
}

func (sched *Scheduler) ScheduleBab() {
	plugins := sched.config.PluginSet
	// Remove all plugin context data at the beginning of a scheduling cycle.
	if plugins.Data().Ctx != nil {
		plugins.Data().Ctx.Reset()
	}

	//know what nodes we get
	nodes, _ := sched.config.NodeLister.List()
	nodeBook := make(map[string]*kubeNode, len(nodes))
	for index, node := range nodes {
		info := &kubeNode{
			Index:      index,
			Node:       node,
			Scheduling: make([]int, 0),
		}
		nodeBook[node.ObjectMeta.Name] = info
	}
	//this time of v1.pods
	pods := make([]*v1.Pod, 0)
	PopPods := make([]*v1.Pod, 0)
	pendingPods := sched.config.SchedulingQueue.PendingPods()
	if len(pendingPods) == 0 {
		return
	}
	for index := len(pendingPods); index > 0; {
		pod, err := sched.config.SchedulingQueue.Pop()
		PopPods = append(PopPods, pod)
		klog.Infof("%d:%v,pod:%v,pods len %v", index, err, pod.Name, len(PopPods))
		index--
	}
	pods = pendingPods
	klog.Infof("pods slice init:%v", len(pods))

	//for each pod filter nodes and record in nodeInfo
	for index, pod := range pods {
		filterdNodes := make([]*v1.Node, 0)
		filterdNodes = append(filterdNodes, sched.config.Algorithm.PubToPredict(pod, nodes)...)
		klog.Infof("choices of pod:%v, node num:%v", pod.Name, len(filterdNodes))
		if len(filterdNodes) == 0 {
			klog.Info("no fiterNode")
			continue
		}
		for _, fiterdnode := range filterdNodes {
			klog.Infof("index%v of recored node:%v", index, fiterdnode.Name)
			host := nodeBook[fiterdnode.ObjectMeta.Name]
			host.Scheduling = append(host.Scheduling, index)
		}
	}
	//init scheduling pod slice empty
	ScheduleFinishedPod := make(map[string]string, 0)
	//for each node to scheduling
	for _, node := range nodeBook {
		klog.Infof("scheduling node name in nodeBook:%v", node.Node.Name)
		waitToScheduling := make([]*v1.Pod, 0)
		recordPodOrinIndex := make([]*schedulingPod, 0)
		if len(node.Scheduling) == 0 {
			klog.Infof("no schedule of %v", node.Node.Name)
			continue
		}
		klog.Infof("node.Scheduling of %v", node.Scheduling)
		for _, podIndex := range node.Scheduling {
			targetName := pods[podIndex].ObjectMeta.Name
			if _, scheduled := ScheduleFinishedPod[targetName]; scheduled {
				continue
			}
			candicate := &schedulingPod{
				Pod:   pods[podIndex],
				Index: podIndex,
			}
			klog.Infof("candicate pod %v", pods[podIndex].Name)
			waitToScheduling = append(waitToScheduling, pods[podIndex])
			recordPodOrinIndex = append(recordPodOrinIndex, candicate)
		}
		klog.Infof("node schedule start:%v %v", node.Node.Name, len(waitToScheduling))
		result := bab.ScheduleNode(waitToScheduling, node.Node)
		klog.Infof("result pod list:%v", len(result))

		chosenPods := make([]*v1.Pod, 0)
		for _, chosenPodIndex := range result {
			chosenPods = append(chosenPods, recordPodOrinIndex[chosenPodIndex].Pod)
		}

		//set result to exe and next node
		for _, pod := range chosenPods {
			klog.Infof("pod binding:%v", pod.Name)
			plugins = sched.config.PluginSet
			// Remove all plugin context data at the beginning of a scheduling cycle.
			if plugins.Data().Ctx != nil {
				plugins.Data().Ctx.Reset()
			}

			assumedPod := pod.DeepCopy()

			allBound, err := sched.assumeVolumes(assumedPod, node.Node.Name)
			if err != nil {
				klog.Errorf("error assuming volumes: %v", err)
				metrics.PodScheduleErrors.Inc()
				return
			}

			for _, pl := range plugins.ReservePlugins() {
				if err := pl.Reserve(plugins, assumedPod, node.Node.Name); err != nil {
					klog.Errorf("error while running %v reserve plugin for pod %v: %v", pl.Name(), assumedPod.Name, err)
					metrics.PodScheduleErrors.Inc()
					return
				}
			}

			err = sched.assume(assumedPod, node.Node.Name)
			if err != nil {
				klog.Errorf("error assuming pod: %v", err)
				metrics.PodScheduleErrors.Inc()
				return
			}

			go func() {
				// Bind volumes first before Pod
				if !allBound {
					err := sched.bindVolumes(assumedPod)
					if err != nil {
						klog.Errorf("error binding volumes: %v", err)
						metrics.PodScheduleErrors.Inc()
						return
					}
				}

				// Run "prebind" plugins.
				for _, pl := range plugins.PrebindPlugins() {
					approved, err := pl.Prebind(plugins, assumedPod, node.Node.Name)
					if err != nil {
						approved = false
						klog.Errorf("error while running %v prebind plugin for pod %v: %v", pl.Name(), assumedPod.Name, err)
						metrics.PodScheduleErrors.Inc()
					}
					if !approved {
						sched.Cache().ForgetPod(assumedPod)
						var reason string
						if err == nil {
							msg := fmt.Sprintf("prebind plugin %v rejected pod %v.", pl.Name(), assumedPod.Name)
							klog.V(4).Infof(msg)
							err = errors.New(msg)
							reason = v1.PodReasonUnschedulable
						}
						sched.recordSchedulingFailure(assumedPod, err, reason, err.Error())
						return
					}
				}

				err := sched.bind(assumedPod, &v1.Binding{
					ObjectMeta: metav1.ObjectMeta{Namespace: assumedPod.Namespace, Name: assumedPod.Name, UID: assumedPod.UID},
					Target: v1.ObjectReference{
						Kind: "Node",
						Name: node.Node.Name,
					},
				})

				if err != nil {
					klog.Errorf("error binding pod: %v", err)
					metrics.PodScheduleErrors.Inc()
				} else {
					klog.V(2).Infof("pod %v/%v is bound successfully on node %v", assumedPod.Namespace, assumedPod.Name, node.Node.Name)
					metrics.PodScheduleSuccesses.Inc()
				}
			}()
		}
	}
}

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
	//know what nodes we get
	nodes, _ := sched.config.NodeLister.List()
	nodeBook := make(map[string]*kubeNode, len(nodes))
	for index, node := range nodes {
		info := &kubeNode{
			Index:      index,
			Node:       node,
			Scheduling: make([]int, 1),
		}
		nodeBook[node.ObjectMeta.Name] = info
	}
	//this time of v1.pods
	pods := make([]*v1.Pod, 1)
	for pod := sched.config.NextPod(); pod != nil; pod = sched.config.NextPod() {
		pods = append(pods, pod)
	}
	//for each pod filter nodes and record in nodeInfo
	for index, pod := range pods {
		filterdNodes := sched.config.Algorithm.PubToPredict(pod, nodes)
		for _, fiterdnode := range filterdNodes {
			host := nodeBook[fiterdnode.ObjectMeta.Name]
			host.Scheduling = append(host.Scheduling, index)
		}
	}
	//init scheduling pod slice empty
	ScheduleFinishedPod := make(map[string]string, 1)
	//for each node to scheduling
	for _, node := range nodeBook {
		waitToScheduling := make([]*v1.Pod, 1)
		recordPodOrinIndex := make([]*schedulingPod, 1)
		for _, podIndex := range node.Scheduling {
			target := pods[podIndex]
			targetName := target.ObjectMeta.Name
			if _, scheduled := ScheduleFinishedPod[targetName]; scheduled {
				continue
			}
			candicate := &schedulingPod{
				Pod:   target,
				Index: podIndex,
			}
			waitToScheduling = append(waitToScheduling, target)
			recordPodOrinIndex = append(recordPodOrinIndex, candicate)
		}
		result := bab.ScheduleNode(waitToScheduling, node.Node)
		chosenPods := make([]*v1.Pod, 1)
		for _, chosenPodIndex := range result {
			chosenPods = append(chosenPods, recordPodOrinIndex[chosenPodIndex].Pod)
		}
		//set result to exe and next node
		for _, pod := range chosenPods {
			plugins := sched.config.PluginSet
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

/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package health

import (
	"time"

	corev1 "k8s.io/api/core/v1"

	v1 "github.com/dcoppa/karpenter/pkg/apis/v1"
	"github.com/dcoppa/karpenter/pkg/events"
)

func NodeRepairBlocked(node *corev1.Node, nodeClaim *v1.NodeClaim, nodePool *v1.NodePool, reason string) []events.Event {
	return []events.Event{
		{
			InvolvedObject: node,
			Type:           corev1.EventTypeWarning,
			Reason:         "NodeRepairBlocked",
			Message:        reason,
			DedupeValues:   []string{string(node.UID)},
			DedupeTimeout:  time.Minute * 15,
		},
		{
			InvolvedObject: node,
			Type:           corev1.EventTypeWarning,
			Reason:         "NodeRepairBlocked",
			Message:        reason,
			DedupeValues:   []string{string(nodeClaim.UID)},
			DedupeTimeout:  time.Minute * 15,
		},
		{
			InvolvedObject: node,
			Type:           corev1.EventTypeWarning,
			Reason:         "NodeRepairBlocked",
			Message:        reason,
			DedupeValues:   []string{string(nodePool.UID)},
			DedupeTimeout:  time.Minute * 15,
		},
	}
}

func NodeRepairBlockedUnmanagedNodeClaim(node *corev1.Node, nodeClaim *v1.NodeClaim, reason string) []events.Event {
	return []events.Event{
		{
			InvolvedObject: node,
			Type:           corev1.EventTypeWarning,
			Reason:         "NodeRepairBlocked",
			Message:        reason,
			DedupeValues:   []string{string(node.UID)},
			DedupeTimeout:  time.Minute * 15,
		},
		{
			InvolvedObject: node,
			Type:           corev1.EventTypeWarning,
			Reason:         "NodeRepairBlocked",
			Message:        reason,
			DedupeValues:   []string{string(nodeClaim.UID)},
			DedupeTimeout:  time.Minute * 15,
		},
	}
}

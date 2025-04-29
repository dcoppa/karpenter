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

package controllers

import (
	"context"

	"github.com/awslabs/operatorpkg/controller"
	"github.com/awslabs/operatorpkg/status"
	"k8s.io/utils/clock"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	corev1 "k8s.io/api/core/v1"

	v1 "github.com/dcoppa/karpenter/pkg/apis/v1"
	"github.com/dcoppa/karpenter/pkg/cloudprovider"
	"github.com/dcoppa/karpenter/pkg/controllers/disruption"
	"github.com/dcoppa/karpenter/pkg/controllers/disruption/orchestration"
	metricsnode "github.com/dcoppa/karpenter/pkg/controllers/metrics/node"
	metricsnodepool "github.com/dcoppa/karpenter/pkg/controllers/metrics/nodepool"
	metricspod "github.com/dcoppa/karpenter/pkg/controllers/metrics/pod"
	"github.com/dcoppa/karpenter/pkg/controllers/node/health"
	nodehydration "github.com/dcoppa/karpenter/pkg/controllers/node/hydration"
	"github.com/dcoppa/karpenter/pkg/controllers/node/termination"
	"github.com/dcoppa/karpenter/pkg/controllers/node/termination/terminator"
	nodeclaimconsistency "github.com/dcoppa/karpenter/pkg/controllers/nodeclaim/consistency"
	nodeclaimdisruption "github.com/dcoppa/karpenter/pkg/controllers/nodeclaim/disruption"
	"github.com/dcoppa/karpenter/pkg/controllers/nodeclaim/expiration"
	nodeclaimgarbagecollection "github.com/dcoppa/karpenter/pkg/controllers/nodeclaim/garbagecollection"
	nodeclaimhydration "github.com/dcoppa/karpenter/pkg/controllers/nodeclaim/hydration"
	nodeclaimlifecycle "github.com/dcoppa/karpenter/pkg/controllers/nodeclaim/lifecycle"
	podevents "github.com/dcoppa/karpenter/pkg/controllers/nodeclaim/podevents"
	nodepoolcounter "github.com/dcoppa/karpenter/pkg/controllers/nodepool/counter"
	nodepoolhash "github.com/dcoppa/karpenter/pkg/controllers/nodepool/hash"
	nodepoolreadiness "github.com/dcoppa/karpenter/pkg/controllers/nodepool/readiness"
	nodepoolvalidation "github.com/dcoppa/karpenter/pkg/controllers/nodepool/validation"
	"github.com/dcoppa/karpenter/pkg/controllers/provisioning"
	"github.com/dcoppa/karpenter/pkg/controllers/state"
	"github.com/dcoppa/karpenter/pkg/controllers/state/informer"
	"github.com/dcoppa/karpenter/pkg/events"
	"github.com/dcoppa/karpenter/pkg/operator/options"
)

func NewControllers(
	ctx context.Context,
	mgr manager.Manager,
	clock clock.Clock,
	kubeClient client.Client,
	recorder events.Recorder,
	cloudProvider cloudprovider.CloudProvider,
) []controller.Controller {
	cluster := state.NewCluster(clock, kubeClient, cloudProvider)
	p := provisioning.NewProvisioner(kubeClient, recorder, cloudProvider, cluster, clock)
	evictionQueue := terminator.NewQueue(kubeClient, recorder)
	disruptionQueue := orchestration.NewQueue(kubeClient, recorder, cluster, clock, p)

	controllers := []controller.Controller{
		p, evictionQueue, disruptionQueue,
		disruption.NewController(clock, kubeClient, p, cloudProvider, recorder, cluster, disruptionQueue),
		provisioning.NewPodController(kubeClient, p),
		provisioning.NewNodeController(kubeClient, p),
		nodepoolhash.NewController(kubeClient, cloudProvider),
		expiration.NewController(clock, kubeClient, cloudProvider),
		informer.NewDaemonSetController(kubeClient, cluster),
		informer.NewNodeController(kubeClient, cluster),
		informer.NewPodController(kubeClient, cluster),
		informer.NewNodePoolController(kubeClient, cloudProvider, cluster),
		informer.NewNodeClaimController(kubeClient, cloudProvider, cluster),
		termination.NewController(clock, kubeClient, cloudProvider, terminator.NewTerminator(clock, kubeClient, evictionQueue, recorder), recorder),
		metricspod.NewController(kubeClient, cluster),
		metricsnodepool.NewController(kubeClient, cloudProvider),
		metricsnode.NewController(cluster),
		nodepoolreadiness.NewController(kubeClient, cloudProvider),
		nodepoolcounter.NewController(kubeClient, cloudProvider, cluster),
		nodepoolvalidation.NewController(kubeClient, cloudProvider),
		podevents.NewController(clock, kubeClient, cloudProvider),
		nodeclaimconsistency.NewController(clock, kubeClient, cloudProvider, recorder),
		nodeclaimlifecycle.NewController(clock, kubeClient, cloudProvider, recorder),
		nodeclaimgarbagecollection.NewController(clock, kubeClient, cloudProvider),
		nodeclaimdisruption.NewController(clock, kubeClient, cloudProvider),
		nodeclaimhydration.NewController(kubeClient, cloudProvider),
		nodehydration.NewController(kubeClient, cloudProvider),
		status.NewController[*v1.NodeClaim](kubeClient, mgr.GetEventRecorderFor("karpenter"), status.EmitDeprecatedMetrics),
		status.NewController[*v1.NodePool](kubeClient, mgr.GetEventRecorderFor("karpenter"), status.EmitDeprecatedMetrics),
		status.NewGenericObjectController[*corev1.Node](kubeClient, mgr.GetEventRecorderFor("karpenter")),
	}

	// The cloud provider must define status conditions for the node repair controller to use to detect unhealthy nodes
	if len(cloudProvider.RepairPolicies()) != 0 && options.FromContext(ctx).FeatureGates.NodeRepair {
		controllers = append(controllers, health.NewController(kubeClient, cloudProvider, clock, recorder))
	}

	return controllers
}

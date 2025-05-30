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

package lifecycle_test

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	clock "k8s.io/utils/clock/testing"

	"github.com/dcoppa/karpenter/pkg/apis"
	v1 "github.com/dcoppa/karpenter/pkg/apis/v1"
	"github.com/dcoppa/karpenter/pkg/cloudprovider/fake"
	nodeclaimlifecycle "github.com/dcoppa/karpenter/pkg/controllers/nodeclaim/lifecycle"
	"github.com/dcoppa/karpenter/pkg/events"
	"github.com/dcoppa/karpenter/pkg/operator/options"
	"github.com/dcoppa/karpenter/pkg/test"
	. "github.com/dcoppa/karpenter/pkg/test/expectations"
	"github.com/dcoppa/karpenter/pkg/test/v1alpha1"
	. "github.com/dcoppa/karpenter/pkg/utils/testing"
)

var ctx context.Context
var nodeClaimController *nodeclaimlifecycle.Controller
var env *test.Environment
var fakeClock *clock.FakeClock
var cloudProvider *fake.CloudProvider

func TestAPIs(t *testing.T) {
	ctx = TestContextWithLogger(t)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Lifecycle")
}

var _ = BeforeSuite(func() {
	fakeClock = clock.NewFakeClock(time.Now())
	env = test.NewEnvironment(test.WithCRDs(apis.CRDs...), test.WithCRDs(v1alpha1.CRDs...), test.WithFieldIndexers(test.NodeProviderIDFieldIndexer(ctx)))
	ctx = options.ToContext(ctx, test.Options())

	cloudProvider = fake.NewCloudProvider()
	nodeClaimController = nodeclaimlifecycle.NewController(fakeClock, env.Client, cloudProvider, events.NewRecorder(&record.FakeRecorder{}))
})

var _ = AfterSuite(func() {
	Expect(env.Stop()).To(Succeed(), "Failed to stop environment")
})

var _ = AfterEach(func() {
	fakeClock.SetTime(time.Now())
	ExpectCleanedUp(ctx, env.Client)
	cloudProvider.Reset()
})

var _ = Describe("Finalizer", func() {
	var nodePool *v1.NodePool

	BeforeEach(func() {
		nodePool = test.NodePool()
	})
	Context("TerminationFinalizer", func() {
		It("should add the finalizer if it doesn't exist", func() {
			nodeClaim := test.NodeClaim(v1.NodeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						v1.NodePoolLabelKey: nodePool.Name,
					},
				},
			})
			ExpectApplied(ctx, env.Client, nodePool, nodeClaim)
			ExpectObjectReconciled(ctx, env.Client, nodeClaimController, nodeClaim)

			nodeClaim = ExpectExists(ctx, env.Client, nodeClaim)
			_, ok := lo.Find(nodeClaim.Finalizers, func(f string) bool {
				return f == v1.TerminationFinalizer
			})
			Expect(ok).To(BeTrue())
		})
		It("shouldn't add the finalizer to NodeClaims not managed by this instance of Karpenter", func() {
			nodeClaim := test.NodeClaim(v1.NodeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						v1.NodePoolLabelKey: nodePool.Name,
					},
				},
				Spec: v1.NodeClaimSpec{
					NodeClassRef: &v1.NodeClassReference{
						Group: "karpenter.test.sh",
						Kind:  "UnmanagedNodeClass",
						Name:  "default",
					},
				},
			})
			ExpectApplied(ctx, env.Client, nodePool, nodeClaim)
			ExpectObjectReconciled(ctx, env.Client, nodeClaimController, nodeClaim)

			nodeClaim = ExpectExists(ctx, env.Client, nodeClaim)
			_, ok := lo.Find(nodeClaim.Finalizers, func(f string) bool {
				return f == v1.TerminationFinalizer
			})
			Expect(ok).To(BeFalse())
		})
	})
})

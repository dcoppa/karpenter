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

package main

import (
	"sigs.k8s.io/controller-runtime/pkg/log"

	kwok "github.com/dcoppa/karpenter/kwok/cloudprovider"
	"github.com/dcoppa/karpenter/pkg/controllers"
	"github.com/dcoppa/karpenter/pkg/operator"
)

func main() {
	ctx, op := operator.NewOperator()
	instanceTypes, err := kwok.ConstructInstanceTypes()
	if err != nil {
		log.FromContext(ctx).Error(err, "failed constructing instance types")
	}

	cloudProvider := kwok.NewCloudProvider(ctx, op.GetClient(), instanceTypes)
	op.
		WithControllers(ctx, controllers.NewControllers(
			ctx,
			op.Manager,
			op.Clock,
			op.GetClient(),
			op.EventRecorder,
			cloudProvider,
		)...).Start(ctx)
}

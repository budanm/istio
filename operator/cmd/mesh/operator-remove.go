// Copyright Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mesh

import (
	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"istio.io/istio/operator/pkg/helmreconciler"
	"istio.io/istio/operator/pkg/name"
	"istio.io/istio/operator/pkg/util/clog"
)

type operatorRemoveArgs struct {
	// kubeConfigPath is the path to kube config file.
	kubeConfigPath string
	// context is the cluster context in the kube config.
	context string
	// force proceeds even if there are validation errors
	force bool
	// operatorNamespace is the namespace the operator controller is installed into.
	operatorNamespace string
}

func addOperatorRemoveFlags(cmd *cobra.Command, oiArgs *operatorRemoveArgs) {
	cmd.PersistentFlags().StringVarP(&oiArgs.kubeConfigPath, "kubeconfig", "c", "", "Path to kube config")
	cmd.PersistentFlags().StringVar(&oiArgs.context, "context", "", "The name of the kubeconfig context to use")
	cmd.PersistentFlags().BoolVar(&oiArgs.force, "force", false, "Proceed even with errors")
	cmd.PersistentFlags().StringVar(&oiArgs.operatorNamespace, "operatorNamespace", operatorDefaultNamespace,
		"The namespace the operator controller is installed into")
}

func operatorRemoveCmd(rootArgs *rootArgs, orArgs *operatorRemoveArgs) *cobra.Command {
	return &cobra.Command{
		Use:   "remove",
		Short: "Removes the Istio operator controller from the cluster.",
		Long:  "The remove subcommand removes the Istio operator controller from the cluster.",
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			l := clog.NewConsoleLogger(cmd.OutOrStdout(), cmd.OutOrStderr(), installerScope)
			operatorRemove(rootArgs, orArgs, l)
		}}
}

// operatorRemove removes the Istio operator controller from the cluster.
func operatorRemove(args *rootArgs, orArgs *operatorRemoveArgs, l clog.Logger) {
	initLogsOrExit(args)

	kubeClient, err := k8sClient(orArgs.kubeConfigPath, orArgs.context)
	if err != nil {
		l.LogAndFatal(err)
	}

	installed, err := isControllerInstalled(kubeClient.Kube(), orArgs.operatorNamespace)
	if installed && err != nil {
		l.LogAndFatal(err)
	}
	if !installed {
		l.LogAndPrintf("Operator controller is not installed in %s namespace (no Deployment detected).", orArgs.operatorNamespace)
		if !orArgs.force {
			l.LogAndFatal("Aborting, use --force to override.")
		}
	}

	l.LogAndPrintf("Removing Istio operator...")
	reconciler, err := helmreconciler.NewHelmReconciler(kubeClient.Controller(), kubeClient.RESTConfig(), nil,
		&helmreconciler.Options{DryRun: args.dryRun, Log: l})
	if err != nil {
		l.LogAndFatal(err)
	}
	if err := reconciler.DeleteComponent(string(name.IstioOperatorComponentName)); err != nil {
		l.LogAndFatal(err)
	}
	if err := deleteNamespace(kubeClient.Kube(), orArgs.operatorNamespace); err != nil {
		l.LogAndFatal(err)
	}
	l.LogAndPrint("Deleted namespace " + orArgs.operatorNamespace)

	l.LogAndPrint(color.New(color.FgGreen).Sprint("✔ ") + "Removal complete")
}

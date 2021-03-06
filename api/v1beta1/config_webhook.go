/*
Copyright 2022 kubetrail.io authors.

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

package v1beta1

import (
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var configlog = logf.Log.WithName("config-resource")

func (r *Config) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-influxdb-kubetrail-io-v1beta1-config,mutating=true,failurePolicy=fail,sideEffects=None,groups=influxdb.kubetrail.io,resources=configs,verbs=create;update,versions=v1beta1,name=mconfig.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Config{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Config) Default() {
	configlog.Info("default", "name", r.Name)

	if len(r.Spec.OrgName) == 0 {
		r.Spec.OrgName = defaultOrgName
	}

	if len(r.Spec.TokenSecretName) == 0 {
		r.Spec.TokenSecretName = defaultSecretName
	}

	if len(r.Spec.TokenSecretNamespace) == 0 {
		r.Spec.TokenSecretNamespace = r.Namespace
	}

	if len(r.Spec.Addr) == 0 {
		r.Spec.Addr = defaultAddr
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-influxdb-kubetrail-io-v1beta1-config,mutating=false,failurePolicy=fail,sideEffects=None,groups=influxdb.kubetrail.io,resources=configs,verbs=create;update,versions=v1beta1,name=vconfig.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Config{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Config) ValidateCreate() error {
	configlog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Config) ValidateUpdate(old runtime.Object) error {
	configlog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Config) ValidateDelete() error {
	configlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

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
	"fmt"

	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var tokenlog = logf.Log.WithName("token-resource")

func (r *Token) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-influxdb-kubetrail-io-v1beta1-token,mutating=true,failurePolicy=fail,sideEffects=None,groups=influxdb.kubetrail.io,resources=tokens,verbs=create;update,versions=v1beta1,name=mtoken.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Token{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Token) Default() {
	tokenlog.Info("default", "name", r.Name)

	if len(r.Spec.SecretName) == 0 {
		id := uuid.New().String()
		r.Spec.SecretName = fmt.Sprintf("%s-%s", r.Name, id[:5])
	}

	if len(r.Spec.ConfigName) == 0 {
		r.Spec.ConfigName = defaultConfigName
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-influxdb-kubetrail-io-v1beta1-token,mutating=false,failurePolicy=fail,sideEffects=None,groups=influxdb.kubetrail.io,resources=tokens,verbs=create;update,versions=v1beta1,name=vtoken.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Token{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Token) ValidateCreate() error {
	tokenlog.Info("validate create", "name", r.Name)

	for _, permission := range r.Spec.Permissions {
		if _, ok := permissionTypes[permission.PermissionType]; !ok {
			err := fmt.Errorf("invalid permission type")
			tokenlog.Error(err, "invalid permission type")
			return err
		}

		if _, ok := resourceTypes[permission.ResourceType]; !ok {
			err := fmt.Errorf("invalid resource type")
			tokenlog.Error(err, "invalid resource type")
			return err
		}
	}
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Token) ValidateUpdate(old runtime.Object) error {
	tokenlog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Token) ValidateDelete() error {
	tokenlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

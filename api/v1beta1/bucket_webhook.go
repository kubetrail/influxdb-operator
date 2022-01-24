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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var bucketlog = logf.Log.WithName("bucket-resource")

func (r *Bucket) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-influxdb-kubetrail-io-v1beta1-bucket,mutating=true,failurePolicy=fail,sideEffects=None,groups=influxdb.kubetrail.io,resources=buckets,verbs=create;update,versions=v1beta1,name=mbucket.kb.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Bucket{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Bucket) Default() {
	bucketlog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-influxdb-kubetrail-io-v1beta1-bucket,mutating=false,failurePolicy=fail,sideEffects=None,groups=influxdb.kubetrail.io,resources=buckets,verbs=create;update,versions=v1beta1,name=vbucket.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Bucket{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Bucket) ValidateCreate() error {
	bucketlog.Info("validate create", "name", r.Name)

	if r.Spec.SecondsTTL != 0 && r.Spec.SecondsTTL < 3600 {
		err := fmt.Errorf("secondsTtl needs be either 0 or >= 3600")
		bucketlog.Error(err, "bucket spec validation error")
		return err
	}
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Bucket) ValidateUpdate(old runtime.Object) error {
	bucketlog.Info("validate update", "name", r.Name)

	rOld, ok := old.(*Bucket)
	if !ok {
		err := fmt.Errorf("input type assertion error")
		bucketlog.Error(err, "failed to type assert input")
		return err
	}

	if r.Spec.SecondsTTL != rOld.Spec.SecondsTTL ||
		r.Spec.Description != rOld.Spec.Description {
		err := fmt.Errorf("spec fields cannot be updated")
		bucketlog.Error(err, "fields cannot change")
		return err
	}
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Bucket) ValidateDelete() error {
	bucketlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

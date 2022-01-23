package controllers

import (
	"context"
	"errors"
	"fmt"
	"time"

	influxdb "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api/http"
	"github.com/influxdata/influxdb-client-go/v2/domain"
	influxdbv1beta1 "github.com/kubetrail/influxdb-operator/api/v1beta1"
	v1 "k8s.io/api/core/v1"
	apimachineryerrors "k8s.io/apimachinery/pkg/api/errors"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *OrganizationReconciler) FinalizeStatus(ctx context.Context, clientObject client.Object) error {
	if !controllerutil.ContainsFinalizer(clientObject, finalizer) {
		return nil
	}

	reqLogger := log.FromContext(ctx)

	object, ok := clientObject.(*influxdbv1beta1.Organization)
	if !ok {
		err := fmt.Errorf("cientObject to object type assertion error")
		reqLogger.Error(err, "failed to get object instance")
		return err
	}

	// Update the status of the object if not terminating
	if object.Status.Phase != phaseTerminating {
		object.Status = influxdbv1beta1.OrganizationStatus{
			Phase:      phaseTerminating,
			Conditions: object.Status.Conditions,
			Message:    "object is marked for deletion",
			Reason:     reasonObjectMarkedForDeletion,
		}
		if err := r.Status().Update(ctx, object); err != nil {
			reqLogger.Error(err, "failed to update object status")
			return err
		} else {
			reqLogger.Info("updated object status")
			return ObjectUpdated
		}
	}

	return nil
}

func (r *OrganizationReconciler) FinalizeResources(ctx context.Context, clientObject client.Object, req ctrl.Request) error {
	if !controllerutil.ContainsFinalizer(clientObject, finalizer) {
		return nil
	}

	reqLogger := log.FromContext(ctx)

	object, ok := clientObject.(*influxdbv1beta1.Organization)
	if !ok {
		err := fmt.Errorf("cientObject to object type assertion error")
		reqLogger.Error(err, "failed to get object instance")
		return err
	}

	// read config with influxdb info
	config := &influxdbv1beta1.Config{}
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: req.Namespace,
		Name:      object.Spec.ConfigName,
	}, config); err != nil {
		if apimachineryerrors.IsNotFound(err) {
			reqLogger.Info("influxdb config not found, skipping deleting resources")
			return nil
		}
		reqLogger.Error(err, "failed to read influxdb config")
		return err
	}

	// read secret with influxdb token
	secret := &v1.Secret{}
	if err := r.Get(
		ctx,
		types.NamespacedName{
			Namespace: config.Spec.TokenSecretNamespace,
			Name:      config.Spec.TokenSecretName,
		},
		secret,
	); err != nil {
		reqLogger.Error(err, "failed to read influxdb token")
		return err
	}

	newClient := influxdb.NewClient(config.Spec.Addr, string(secret.Data[keyToken]))
	// always close client at the end
	defer newClient.Close()

	orgApi := newClient.OrganizationsAPI()

	organization, err := orgApi.FindOrganizationByName(ctx, object.Name)
	if err != nil {
		httpErr := &http.Error{
			StatusCode: 0,
			Code:       "",
			Message:    "",
			Err:        nil,
			RetryAfter: 0,
		}
		if errors.As(err, &httpErr) && httpErr.StatusCode == 404 {
			reqLogger.Info("organization not found")
			return nil
		} else {
			reqLogger.Error(err, "failed to find organization")
			return err
		}
	}

	if organization == nil || organization.Id == nil || len(*organization.Id) == 0 {
		err := fmt.Errorf("nil org pointer or invalid id")
		reqLogger.Error(err, "failed to get valid org id")
		return err
	}

	if err := orgApi.DeleteOrganizationWithID(ctx, *organization.Id); err != nil {
		reqLogger.Error(err, "failed to delete organization")
		return err
	}

	reqLogger.Info("organization deleted")

	var found bool
	// Update the status of the object if pending
	for i, condition := range object.Status.Conditions {
		if condition.Reason == reasonDeletedOrganization {
			object.Status.Conditions[i].LastTransitionTime = v12.Time{Time: time.Now()}
			found = true
			break
		}
	}

	if !found {
		condition := v12.Condition{
			Type:               conditionTypeInfluxdb,
			Status:             v12.ConditionTrue,
			ObservedGeneration: 0,
			LastTransitionTime: v12.Time{Time: time.Now()},
			Reason:             reasonDeletedOrganization,
			Message:            "deleted influxdb organization",
		}
		object.Status = influxdbv1beta1.OrganizationStatus{
			Phase:      object.Status.Phase,
			Conditions: append(object.Status.Conditions, condition),
			Message:    "deleted influxdb organization",
			Reason:     reasonDeletedOrganization,
		}
	}

	if err := r.Status().Update(ctx, object); err != nil {
		reqLogger.Error(err, "failed to update object status")
		return err
	} else {
		reqLogger.Info("updated object status")
		return ObjectUpdated
	}
}

func (r *OrganizationReconciler) RemoveFinalizer(ctx context.Context, clientObject client.Object) error {
	if !controllerutil.ContainsFinalizer(clientObject, finalizer) {
		return nil
	}

	reqLogger := log.FromContext(ctx)

	controllerutil.RemoveFinalizer(clientObject, finalizer)
	if err := r.Update(ctx, clientObject); err != nil {
		reqLogger.Error(err, "failed to remove finalizer")
		return err
	}
	reqLogger.Info("finalizer removed")
	return ObjectUpdated
}

func (r *OrganizationReconciler) AddFinalizer(ctx context.Context, clientObject client.Object) error {
	if controllerutil.ContainsFinalizer(clientObject, finalizer) {
		return nil
	}

	reqLogger := log.FromContext(ctx)

	controllerutil.AddFinalizer(clientObject, finalizer)
	if err := r.Update(ctx, clientObject); err != nil {
		reqLogger.Error(err, "failed to add finalizer")
		return err
	}
	reqLogger.Info("finalizer added")
	return ObjectUpdated
}

func (r *OrganizationReconciler) InitializeStatus(ctx context.Context, clientObject client.Object) error {
	reqLogger := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(clientObject, finalizer) {
		err := fmt.Errorf("finalizer not found")
		reqLogger.Error(err, "failed to detect finalizer")
		return err
	}

	object, ok := clientObject.(*influxdbv1beta1.Organization)
	if !ok {
		err := fmt.Errorf("cientObject to object type assertion error")
		reqLogger.Error(err, "failed to get object instance")
		return err
	}

	// Update the status of the object if none exists
	found := false
	for _, condition := range object.Status.Conditions {
		if condition.Reason == reasonFinalizerAdded {
			found = true
			break
		}
	}

	if !found {
		object.Status = influxdbv1beta1.OrganizationStatus{
			Phase: phasePending,
			Conditions: []v12.Condition{
				{
					Type:               conditionTypeObject,
					Status:             v12.ConditionTrue,
					ObservedGeneration: 0,
					LastTransitionTime: v12.Time{Time: time.Now()},
					Reason:             reasonFinalizerAdded,
					Message:            "object initialized",
				},
			},
			Message: "object initialized",
			Reason:  reasonObjectInitialized,
		}
		if err := r.Status().Update(ctx, object); err != nil {
			reqLogger.Error(err, "failed to update object status")
			return err
		} else {
			reqLogger.Info("updated object status")
			return ObjectUpdated
		}
	}

	return nil
}

func (r *OrganizationReconciler) ReconcileResources(ctx context.Context, clientObject client.Object, req ctrl.Request) error {
	reqLogger := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(clientObject, finalizer) {
		err := fmt.Errorf("finalizer not found")
		reqLogger.Error(err, "failed to detect finalizer")
		return err
	}

	object, ok := clientObject.(*influxdbv1beta1.Organization)
	if !ok {
		err := fmt.Errorf("cientObject to object type assertion error")
		reqLogger.Error(err, "failed to get object instance")
		return err
	}

	// read config with influxdb info
	config := &influxdbv1beta1.Config{}
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: req.Namespace,
		Name:      object.Spec.ConfigName,
	}, config); err != nil {
		reqLogger.Error(err, "failed to read influxdb config")
		return err
	}

	// read secret with influxdb token
	secret := &v1.Secret{}
	if err := r.Get(
		ctx,
		types.NamespacedName{
			Namespace: config.Spec.TokenSecretNamespace,
			Name:      config.Spec.TokenSecretName,
		},
		secret,
	); err != nil {
		reqLogger.Error(err, "failed to read influxdb token")
		return err
	}

	newClient := influxdb.NewClient(config.Spec.Addr, string(secret.Data[keyToken]))
	// always close client at the end
	defer newClient.Close()

	orgApi := newClient.OrganizationsAPI()

	organization, err := orgApi.FindOrganizationByName(ctx, config.Spec.OrgName)
	if err != nil || organization == nil || organization.Id == nil {
		reqLogger.Error(err, "failed to find influxdb org")
		return err
	}

	var organizationCreated bool
	var found bool

	if _, err := orgApi.CreateOrganization(
		ctx,
		&domain.Organization{
			CreatedAt:   nil,
			Description: nil,
			Id:          nil,
			Links:       nil,
			Name:        object.Name,
			Status:      nil,
			UpdatedAt:   nil,
		},
	); err != nil {
		httpErr := &http.Error{
			StatusCode: 0,
			Code:       "",
			Message:    "",
			Err:        nil,
			RetryAfter: 0,
		}
		if errors.As(err, &httpErr) && httpErr.StatusCode == 422 {
			rateLimit(
				fmt.Sprintf("%s-%s", object.UID, "org"),
				time.Hour*24,
				func() {
					reqLogger.Info("org exists")
				},
			)
		} else {
			reqLogger.Error(err, "failed to create organization")
			return err
		}
	} else {
		reqLogger.Info("organization created")
		organizationCreated = true
	}

	// Update the status of the object if pending
	for i, condition := range object.Status.Conditions {
		if condition.Reason == reasonCreatedOrganization {
			if organizationCreated {
				object.Status.Conditions[i].LastTransitionTime = v12.Time{Time: time.Now()}
			}
			found = true
			break
		}
	}
	if !found {
		condition := v12.Condition{
			Type:               conditionTypeInfluxdb,
			Status:             v12.ConditionTrue,
			ObservedGeneration: 0,
			LastTransitionTime: v12.Time{Time: time.Now()},
			Reason:             reasonCreatedOrganization,
			Message:            "created influxdb organization",
		}
		object.Status = influxdbv1beta1.OrganizationStatus{
			Phase:      phaseReady,
			Conditions: append(object.Status.Conditions, condition),
			Message:    "created influxdb organization",
			Reason:     reasonCreatedOrganization,
		}
		if err := r.Status().Update(ctx, object); err != nil {
			reqLogger.Error(err, "failed to update object status")
			return err
		} else {
			reqLogger.Info("updated object status")
			return ObjectUpdated
		}
	} else {
		if organizationCreated {
			if err := r.Status().Update(ctx, object); err != nil {
				reqLogger.Error(err, "failed to update object status")
				return err
			} else {
				reqLogger.Info("updated object status")
				return ObjectUpdated
			}
		}
	}

	return nil
}

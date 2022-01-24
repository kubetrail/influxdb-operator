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

func (r *TokenReconciler) FinalizeStatus(ctx context.Context, clientObject client.Object) error {
	if !controllerutil.ContainsFinalizer(clientObject, finalizer) {
		return nil
	}

	reqLogger := log.FromContext(ctx)

	object, ok := clientObject.(*influxdbv1beta1.Token)
	if !ok {
		err := fmt.Errorf("cientObject to object type assertion error")
		reqLogger.Error(err, "failed to get object instance")
		return err
	}

	// Update the status of the object if not terminating
	if object.Status.Phase != phaseTerminating {
		object.Status = influxdbv1beta1.TokenStatus{
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

func (r *TokenReconciler) FinalizeResources(ctx context.Context, clientObject client.Object, req ctrl.Request) error {
	if !controllerutil.ContainsFinalizer(clientObject, finalizer) {
		return nil
	}

	reqLogger := log.FromContext(ctx)

	object, ok := clientObject.(*influxdbv1beta1.Token)
	if !ok {
		err := fmt.Errorf("cientObject to object type assertion error")
		reqLogger.Error(err, "failed to get object instance")
		return err
	}

	var tokenId string
	if object.Status.Data != nil {
		if id, ok := object.Status.Data[keyTokenId]; ok {
			tokenId = id
		}
	}

	// influxdb auth token description that is used to identify the token later
	authorizationDescription := getAuthorizationDescription(object.Name, object.Namespace, string(object.UID))

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

	organization, err := orgApi.FindOrganizationByName(ctx, config.Spec.OrgName)
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

	authorizationsApi := newClient.AuthorizationsAPI()

	var found bool
	authorizations, err := authorizationsApi.FindAuthorizationsByOrgID(ctx, *organization.Id)
	if err != nil {
		reqLogger.Error(err, "failed to find tokens by org id")
		return err
	}

	if authorizations == nil {
		err := fmt.Errorf("recived nil authorizations")
		reqLogger.Error(err, "failed to get valid authorizations")
		return err
	}

	for _, authorization := range *authorizations {
		if authorization.Description != nil &&
			*authorization.Description == authorizationDescription {
			tokenId = *authorization.Id
			found = true
			break
		}
	}

	if !found {
		reqLogger.Info("token not found")
		return nil
	}

	if err := authorizationsApi.DeleteAuthorizationWithID(ctx, tokenId); err != nil {
		reqLogger.Error(err, "failed to delete token")
		return err
	}

	reqLogger.Info("token deleted")

	found = false
	// Update the status of the object if pending
	for i, condition := range object.Status.Conditions {
		if condition.Reason == reasonDeletedToken {
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
			Reason:             reasonDeletedToken,
			Message:            "deleted influxdb token",
		}
		object.Status = influxdbv1beta1.TokenStatus{
			Phase:      object.Status.Phase,
			Conditions: append(object.Status.Conditions, condition),
			Message:    "deleted influxdb token",
			Reason:     reasonDeletedToken,
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

func (r *TokenReconciler) RemoveFinalizer(ctx context.Context, clientObject client.Object) error {
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

func (r *TokenReconciler) AddFinalizer(ctx context.Context, clientObject client.Object) error {
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

func (r *TokenReconciler) InitializeStatus(ctx context.Context, clientObject client.Object) error {
	reqLogger := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(clientObject, finalizer) {
		err := fmt.Errorf("finalizer not found")
		reqLogger.Error(err, "failed to detect finalizer")
		return err
	}

	object, ok := clientObject.(*influxdbv1beta1.Token)
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
		object.Status = influxdbv1beta1.TokenStatus{
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

func (r *TokenReconciler) ReconcileResources(ctx context.Context, clientObject client.Object, req ctrl.Request) error {
	reqLogger := log.FromContext(ctx)

	var tokenExists bool
	var secretExists bool
	var tokenCreated bool
	var found bool
	var tokenId string
	var token string

	if !controllerutil.ContainsFinalizer(clientObject, finalizer) {
		err := fmt.Errorf("finalizer not found")
		reqLogger.Error(err, "failed to detect finalizer")
		return err
	}

	object, ok := clientObject.(*influxdbv1beta1.Token)
	if !ok {
		err := fmt.Errorf("cientObject to object type assertion error")
		reqLogger.Error(err, "failed to get object instance")
		return err
	}

	if object.Status.Data != nil {
		if id, ok := object.Status.Data[keyTokenId]; ok {
			tokenId = id
		}
	}

	// influxdb auth token description that is used to identify the token later
	authorizationDescription := getAuthorizationDescription(object.Name, object.Namespace, string(object.UID))

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
	if err != nil {
		httpErr := &http.Error{
			StatusCode: 0,
			Code:       "",
			Message:    "",
			Err:        nil,
			RetryAfter: 0,
		}
		if errors.As(err, &httpErr) && httpErr.StatusCode == 404 {
			reqLogger.Error(err, "organization not found")
			return err
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

	authorizationsApi := newClient.AuthorizationsAPI()

	authorizations, err := authorizationsApi.FindAuthorizationsByOrgID(ctx, *organization.Id)
	if err != nil {
		reqLogger.Error(err, "failed to find tokens by org id")
		return err
	}

	if authorizations == nil {
		err := fmt.Errorf("recived nil authorizations")
		reqLogger.Error(err, "failed to get valid authorizations")
		return err
	}

	for _, authorization := range *authorizations {
		if authorization.Description != nil &&
			*authorization.Description == authorizationDescription {
			rateLimit(
				fmt.Sprintf("%s-%s", object.UID, "token"),
				time.Hour*24,
				func() {
					reqLogger.Info("token exists")
				},
			)
			tokenExists = true

			if authorization.Id == nil || authorization.Token == nil {
				err := fmt.Errorf("received nil id or token in authorization")
				reqLogger.Error(err, "failed to get valid authorization")
				return err
			}

			tokenId = *authorization.Id
			token = *authorization.Token
			break
		}
	}

	if !tokenExists {
		permissions := make([]domain.Permission, len(object.Spec.Permissions))
		for i, permission := range object.Spec.Permissions {
			permission := permission
			name := new(string)
			if len(permission.ResourceName) > 0 {
				*name = permission.ResourceName
			}
			if permission.ResourceType == influxdbv1beta1.ResourceTypeOrgs {
				permissions[i] = domain.Permission{
					Action: domain.PermissionAction(permission.PermissionType),
					Resource: domain.Resource{
						Id:    nil,
						Name:  name,
						Org:   nil,
						OrgID: nil,
						Type:  domain.ResourceType(permission.ResourceType),
					},
				}
			} else {
				permissions[i] = domain.Permission{
					Action: domain.PermissionAction(permission.PermissionType),
					Resource: domain.Resource{
						Id:    nil,
						Name:  name,
						Org:   &organization.Name,
						OrgID: organization.Id,
						Type:  domain.ResourceType(permission.ResourceType),
					},
				}
			}
		}

		if authorization, err := authorizationsApi.CreateAuthorization(
			ctx,
			&domain.Authorization{
				AuthorizationUpdateRequest: domain.AuthorizationUpdateRequest{
					Description: &authorizationDescription,
					Status:      nil,
				},
				CreatedAt:   nil,
				Id:          nil,
				Links:       nil,
				Org:         nil,
				OrgID:       organization.Id,
				Permissions: &permissions,
				Token:       nil,
				UpdatedAt:   nil,
				User:        nil,
				UserID:      nil,
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
					fmt.Sprintf("%s-%s", object.UID, "token"),
					time.Hour*24,
					func() {
						reqLogger.Info("token exists")
					},
				)
				tokenId = *authorization.Id
				token = *authorization.Token
			} else {
				reqLogger.Error(err, "failed to create token")
				return err
			}
		} else {
			reqLogger.Info("token created")
			tokenCreated = true
			tokenId = *authorization.Id
			token = *authorization.Token
		}
	}

	if tokenExists || tokenCreated {
		secret := &v1.Secret{
			TypeMeta: v12.TypeMeta{},
			ObjectMeta: v12.ObjectMeta{
				Name:                       object.Spec.SecretName,
				GenerateName:               "",
				Namespace:                  object.Namespace,
				SelfLink:                   "",
				UID:                        "",
				ResourceVersion:            "",
				Generation:                 0,
				CreationTimestamp:          v12.Time{Time: time.Now()},
				DeletionTimestamp:          nil,
				DeletionGracePeriodSeconds: nil,
				Labels:                     nil,
				Annotations:                nil,
				OwnerReferences: []v12.OwnerReference{
					{
						APIVersion:         object.APIVersion,
						Kind:               object.Kind,
						Name:               object.Name,
						UID:                object.UID,
						Controller:         nil,
						BlockOwnerDeletion: nil,
					},
				},
				Finalizers:    nil,
				ClusterName:   "",
				ManagedFields: nil,
			},
			Immutable: nil,
			Data: map[string][]byte{
				keyToken: []byte(token),
			},
			StringData: nil,
			Type:       "",
		}

		if err := r.Create(ctx, secret); err != nil {
			if apimachineryerrors.IsAlreadyExists(err) {
				rateLimit(
					fmt.Sprintf("%s-%s", object.UID, "secret"),
					time.Hour*24,
					func() {
						reqLogger.Info("secret exists")
					},
				)
				secretExists = true
			} else {
				reqLogger.Error(err, "failed to create secret")
				return err
			}
		} else {
			reqLogger.Info("created secret")
		}
	}

	// update secret if already exists but does not contain
	// data matching with token
	if secretExists {
		secret := &v1.Secret{}
		if err := r.Get(ctx, types.NamespacedName{
			Namespace: object.Namespace,
			Name:      object.Spec.SecretName,
		}, secret); err != nil {
			reqLogger.Error(err, "failed to get secret")
			return err
		}

		if secret.Data == nil || string(secret.Data[keyToken]) != token {
			secret.Data = map[string][]byte{
				keyToken: []byte(token),
			}

			if err := r.Update(ctx, secret); err != nil {
				reqLogger.Error(err, "failed to update secret")
				return err
			} else {
				reqLogger.Info("updated secret")
			}
		}
	}

	// Update the status of the object if pending
	for i, condition := range object.Status.Conditions {
		if condition.Reason == reasonCreatedToken {
			if tokenCreated {
				object.Status.Conditions[i].LastTransitionTime = v12.Time{Time: time.Now()}
				object.Status.Data = map[string]string{
					keyTokenId: tokenId,
				}
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
			Reason:             reasonCreatedToken,
			Message:            "created influxdb token",
		}
		object.Status = influxdbv1beta1.TokenStatus{
			Phase:      phaseReady,
			Conditions: append(object.Status.Conditions, condition),
			Message:    "created influxdb token",
			Reason:     reasonCreatedToken,
		}
		if err := r.Status().Update(ctx, object); err != nil {
			reqLogger.Error(err, "failed to update object status")
			return err
		} else {
			reqLogger.Info("updated object status")
			return ObjectUpdated
		}
	} else {
		if tokenCreated {
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

func getAuthorizationDescription(name, namespace, uid string) string {
	return fmt.Sprintf("%s.%s.%s", name, namespace, uid)
}

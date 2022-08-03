/*
Copyright 2022.

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
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	deploymentv1 "github.com/Madongming/move-clouds-deployment/api/v1"
)

// SingleDeploymentReconciler reconciles a SingleDeployment object
type SingleDeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=deployment.github.com,resources=singledeployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=deployment.github.com,resources=singledeployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=deployment.github.com,resources=singledeployments/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="networking.k8s.io",resources=ingresses,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the SingleDeployment object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *SingleDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx, "SingleDeployment", req.NamespacedName)

	// Get SingleDeployment object
	sd := new(deploymentv1.SingleDeployment)
	if err := r.Client.Get(ctx, req.NamespacedName, sd); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Deep-copy single deployment otherwise we are mutating our cache
	sdCopy := sd.DeepCopy()

	// Watch and create/update deployment
	///////////////////////////////////////////////////////////////
	deployment := &appsv1.Deployment{}
	if err := r.Client.Get(ctx, req.NamespacedName, deployment); err != nil {
		if errors.IsNotFound(err) {
			// Its a "not found error" that is none a deployment, create it.

			// Create deployment
			if errCreate := r.createDeployment(ctx, logger, sdCopy); errCreate != nil {
				// create failed
				r.setConditions(
					&sdCopy.Status,
					deploymentv1.ConditionTypeDeployment,
					sdCopy.Name,
					fmt.Sprintf("Deployment \"%s\" create failed: %s", sdCopy.Name, errCreate.Error()),
					deploymentv1.ConditionStatusFailed,
					deploymentv1.ConditionReasonDeploymentUnavailable,
				)
			} else {
				r.setConditions(
					&sdCopy.Status,
					deploymentv1.ConditionTypeDeployment,
					sdCopy.Name,
					fmt.Sprintf("Deployment \"%s\" is creating", sdCopy.Name),
					deploymentv1.ConditionStatusUnKnown,
					deploymentv1.ConditionReasonDeploymentUnavailable,
				)
			}
		} else {
			// Its not a "not found err", throw it
			logger.Error(err, "Get deployment failed")
			r.setConditions(
				&sdCopy.Status,
				deploymentv1.ConditionTypeDeployment,
				sdCopy.Name,
				fmt.Sprintf("Deployment \"%s\" get failed: %s", sdCopy.Name, err.Error()),
				deploymentv1.ConditionStatusFailed,
				deploymentv1.ConditionReasonDeploymentUnavailable,
			)
		}
	} else {
		// Exist the deployment, update it

		// Update deployment, include status
		if err := r.updateDeployment(ctx, logger, sdCopy, deployment); err != nil {
			// update failed
			logger.Error(err, "Update Deployment failed")
			r.setConditions(
				&sdCopy.Status,
				deploymentv1.ConditionTypeDeployment,
				sdCopy.Name,
				fmt.Sprintf("Deployment \"%s\" is update failed: %s", sdCopy.Name, err.Error()),
				deploymentv1.ConditionStatusFailed,
				deploymentv1.ConditionReasonDeploymentUnavailable,
			)
			// Sync deployment status to singledeployment
		} else if deployment.Status.AvailableReplicas == sdCopy.Spec.Replicas {
			r.setConditions(
				&sdCopy.Status,
				deploymentv1.ConditionTypeDeployment,
				sdCopy.Name,
				fmt.Sprintf("Deployment \"%s\" is created", sdCopy.Name),
				deploymentv1.ConditionStatusReady,
				deploymentv1.ConditionReasonDeploymentAvailable,
			)
		} else {
			r.setConditions(
				&sdCopy.Status,
				deploymentv1.ConditionTypeDeployment,
				sdCopy.Name,
				fmt.Sprintf("Deployment \"%s\" is creating", sdCopy.Name),
				deploymentv1.ConditionStatusUnKnown,
				deploymentv1.ConditionReasonDeploymentUnavailable,
			)
		}
	}
	///////////////////////////////////////////////////////////////

	// Ingress mode or NodePort mode
	// Watch and create/update Service and Ingress
	///////////////////////////////////////////////////////////////
	// Deal with Service
	service := new(corev1.Service)
	if err := r.Client.Get(ctx, req.NamespacedName, service); err != nil {
		if errors.IsNotFound(err) {
			// Its a "not found error" that is none a service, create it.
			// Create service
			if errCreate := r.createService(ctx, logger, sdCopy); errCreate != nil {
				// create failed
				logger.Error(errCreate, "Create Service failed")
				r.setConditions(
					&sdCopy.Status,
					deploymentv1.ConditionTypeService,
					sdCopy.Name,
					fmt.Sprintf("Service \"%s\" is create failed: %s", sdCopy.Name, errCreate.Error()),
					deploymentv1.ConditionStatusFailed,
					deploymentv1.ConditionReasonServiceUnavailable,
				)
			} else {
				// if Service create / update call is success,it is alway created successful
				r.setConditions(
					&sdCopy.Status,
					deploymentv1.ConditionTypeService,
					sdCopy.Name,
					fmt.Sprintf("Service \"%s\" is created", sdCopy.Name),
					deploymentv1.ConditionStatusReady,
					deploymentv1.ConditionReasonServiceAvailable,
				)
			}
		} else {
			// Its not a "not found err", throw it
			logger.Error(err, "Get service failed")
			r.setConditions(
				&sdCopy.Status,
				deploymentv1.ConditionTypeService,
				sdCopy.Name,
				fmt.Sprintf("Service \"%s\" is get failed: %s", sdCopy.Name, err.Error()),
				deploymentv1.ConditionStatusFailed,
				deploymentv1.ConditionReasonServiceUnavailable,
			)
		}
	} else {
		// The service is exist update the service

		// Update service
		if err := r.updateService(ctx, logger, sdCopy, service); err != nil {
			// update failed
			logger.Error(err, "update Service failed")
			r.setConditions(
				&sdCopy.Status,
				deploymentv1.ConditionTypeService,
				sdCopy.Name,
				fmt.Sprintf("Service \"%s\" is update failed: %s", sdCopy.Name, err.Error()),
				deploymentv1.ConditionStatusFailed,
				deploymentv1.ConditionReasonServiceUnavailable,
			)
		} else {
			// if Service create / update call is success,it is alway created successful,
			r.setConditions(
				&sdCopy.Status,
				deploymentv1.ConditionTypeService,
				sdCopy.Name,
				fmt.Sprintf("Service \"%s\" is created", sdCopy.Name),
				deploymentv1.ConditionStatusReady,
				deploymentv1.ConditionReasonServiceAvailable,
			)
		}
	}

	// Deal with Ingress
	ingress := new(netv1.Ingress)
	if err := r.Client.Get(ctx, req.NamespacedName, ingress); err != nil {
		if errors.IsNotFound(err) {
			// Its a "not found error" that is not exsit an ingress, create it.
			if strings.ToLower(sdCopy.Spec.Expose.Mode) == "ingress" {
				// It is ingress mode, Create ingress
				if errCreate := r.createIngress(ctx, logger, sdCopy); errCreate != nil {
					// create failed
					logger.Error(err, "Create Ingress failed")
					r.setConditions(
						&sdCopy.Status,
						deploymentv1.ConditionTypeIngress,
						sdCopy.Name,
						fmt.Sprintf("Ingress \"%s\" is create failed: %s", sdCopy.Name, errCreate.Error()),
						deploymentv1.ConditionStatusFailed,
						deploymentv1.ConditionReasonServiceUnavailable,
					)
				}
			}
		} else {
			// Its not a "not found err", throw it
			logger.Error(err, "create ingress failed")
			r.setConditions(
				&sdCopy.Status,
				deploymentv1.ConditionTypeIngress,
				sdCopy.Name,
				fmt.Sprintf("Ingress \"%s\" is get failed: %s", sdCopy.Name, err.Error()),
				deploymentv1.ConditionStatusFailed,
				deploymentv1.ConditionReasonServiceUnavailable,
			)
		}
	} else {
		if strings.ToLower(sd.Spec.Expose.Mode) == "ingress" {
			// The ingress is exist and expose mode is ingress, update the ingress

			// Update ingress

			if err := r.updateIngress(ctx, logger, sd, ingress); err != nil {
				// update failed
				logger.Error(err, "update Ingress failed")
				r.setConditions(
					&sdCopy.Status,
					deploymentv1.ConditionTypeIngress,
					sdCopy.Name,
					fmt.Sprintf("Ingress \"%s\" is update failed: %s", sdCopy.Name, err.Error()),
					deploymentv1.ConditionStatusFailed,
					deploymentv1.ConditionReasonServiceUnavailable,
				)
			} else {
				// if Ingress create / update call is success,it is alway created successful,
				r.setConditions(
					&sdCopy.Status,
					deploymentv1.ConditionTypeIngress,
					sdCopy.Name,
					fmt.Sprintf("Ingress \"%s\" is created", sdCopy.Name),
					deploymentv1.ConditionStatusReady,
					deploymentv1.ConditionReasonIngressAvailable,
				)
			}
		} else if strings.ToLower(sd.Spec.Expose.Mode) == "nodeport" {
			// The ingress is exist, but mode is set nodeport, delete the ingress
			// Delete ingress
			if err := r.deleteIngress(ctx, logger, ingress); err != nil {
				// delete failed
				logger.Error(err, "Delete Ingress failed")
				r.setConditions(
					&sdCopy.Status,
					deploymentv1.ConditionTypeIngress,
					sdCopy.Name,
					fmt.Sprintf("Service \"%s\" is update failed: %s", sdCopy.Name, err.Error()),
					deploymentv1.ConditionStatusFailed,
					deploymentv1.ConditionReasonServiceUnavailable,
				)
			} else {
				r.deleteConditions(
					&sdCopy.Status,
					deploymentv1.ConditionTypeIngress,
				)
			}
		}
	}
	///////////////////////////////////////////////////////////////

	// All work is done
	// Judging `status` according to conditions
	r.processStatus(&sdCopy.Status)

	if sd.Status.ObservedGeneration != sdCopy.Status.ObservedGeneration {
		// Some status is changed, update
		if err := r.Client.Status().Update(ctx, sdCopy); err != nil {
			logger.Error(err, "Update status failed")
			return ctrl.Result{RequeueAfter: 10 * time.Second}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SingleDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&deploymentv1.SingleDeployment{}).
		Owns(&appsv1.Deployment{}).
		Owns(&netv1.Ingress{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

// private methods
///////////////////////////////////////////////////////////////
func (r *SingleDeploymentReconciler) generateDeployment(sd *deploymentv1.SingleDeployment) (*appsv1.Deployment, error) {
	deployment, err := newDeployment(sd)
	if err != nil {
		return nil, err
	}
	err = controllerutil.SetControllerReference(sd, deployment, r.Scheme)
	if err != nil {
		return nil, err
	}

	return deployment, nil
}

func (r *SingleDeploymentReconciler) createDeployment(ctx context.Context, logger logr.Logger, sd *deploymentv1.SingleDeployment) error {
	deployment, err := r.generateDeployment(sd)
	if err != nil {
		return err
	}
	if err := r.Client.Create(ctx, deployment); err != nil {
		logger.Error(err, "Create New deployment failed")
		return err
	}

	return nil
}

func (r *SingleDeploymentReconciler) updateDeployment(ctx context.Context, logger logr.Logger, sd *deploymentv1.SingleDeployment, deploy *appsv1.Deployment) error {
	deployment, err := r.generateDeployment(sd)
	if err != nil {
		return err
	}

	if err := r.Client.Update(ctx, deployment, client.DryRunAll); err != nil {
		return err
	}

	if reflect.DeepEqual(deployment.Spec, deploy.Spec) {
		return nil
	}

	if err := r.Client.Update(ctx, deployment); err != nil {
		logger.Error(err, "Update New deployment failed")
		return err
	}

	return nil
}

func (r *SingleDeploymentReconciler) generateService(sd *deploymentv1.SingleDeployment) (*corev1.Service, error) {
	service, err := newService(sd)
	if err != nil {
		return nil, err
	}
	err = controllerutil.SetControllerReference(sd, service, r.Scheme)
	if err != nil {
		return nil, err
	}

	return service, nil
}

func (r *SingleDeploymentReconciler) generateIngress(sd *deploymentv1.SingleDeployment) (*netv1.Ingress, error) {
	ingress, err := newIngress(sd)
	if err != nil {
		return nil, err
	}
	err = controllerutil.SetControllerReference(sd, ingress, r.Scheme)
	if err != nil {
		return nil, err
	}

	return ingress, nil
}

func (r *SingleDeploymentReconciler) createService(ctx context.Context, logger logr.Logger, sd *deploymentv1.SingleDeployment) error {
	service, err := r.generateService(sd)
	if err != nil {
		return err
	}
	if err := r.Client.Create(ctx, service); err != nil {
		logger.Error(err, "Create New service failed")
		return err
	}

	return nil
}

func (r *SingleDeploymentReconciler) updateService(ctx context.Context, logger logr.Logger, sd *deploymentv1.SingleDeployment, svc *corev1.Service) error {
	service, err := r.generateService(sd)
	if err != nil {
		return err
	}

	if err := r.Client.Update(ctx, service, client.DryRunAll); err != nil {
		return err
	}

	if reflect.DeepEqual(service.Spec, svc.Spec) {
		return nil
	}

	if err := r.Client.Update(ctx, service); err != nil {
		logger.Error(err, "Update New service failed")
		return err
	}

	return nil
}

func (r *SingleDeploymentReconciler) createIngress(ctx context.Context, logger logr.Logger, sd *deploymentv1.SingleDeployment) error {
	ingress, err := r.generateIngress(sd)
	if err != nil {
		return err
	}
	if err := r.Client.Create(ctx, ingress); err != nil {
		logger.Error(err, "Create New Ingress failed")
		return err
	}

	return nil
}

func (r *SingleDeploymentReconciler) updateIngress(ctx context.Context, logger logr.Logger, sd *deploymentv1.SingleDeployment, ig *netv1.Ingress) error {
	ingress, err := r.generateIngress(sd)
	if err != nil {
		return err
	}

	if err := r.Client.Update(ctx, ingress, client.DryRunAll); err != nil {
		return err
	}

	if reflect.DeepEqual(ingress.Spec, ig.Spec) {
		return nil
	}

	if err := r.Client.Update(ctx, ingress); err != nil {
		logger.Error(err, "Update New Ingress failed")
		return err
	}

	return nil
}

func (r *SingleDeploymentReconciler) deleteService(ctx context.Context, logger logr.Logger, service *corev1.Service) error {
	if err := r.Client.Delete(ctx, service); err != nil {
		logger.Error(err, "Update New Ingress failed")
		return err
	}
	return nil
}

func (r *SingleDeploymentReconciler) deleteIngress(ctx context.Context, logger logr.Logger, ingress *netv1.Ingress) error {
	if err := r.Client.Delete(ctx, ingress); err != nil {
		logger.Error(err, "Update New Ingress failed")
		return err
	}
	return nil
}

func (r *SingleDeploymentReconciler) setStatus(
	sdStatus *deploymentv1.SingleDeploymentStatus,
	phase,
	message,
	reason string,
) {
	backup := sdStatus.ObservedGeneration

	if sdStatus.Phase != phase {
		sdStatus.Phase = phase
		backup++
	}
	if sdStatus.Message != message {
		sdStatus.Message = message
		backup++
	}
	if sdStatus.Reason != reason {
		sdStatus.Reason = reason
		backup++
	}
	if sdStatus.ObservedGeneration != backup {
		sdStatus.ObservedGeneration++
	}
}

func (r *SingleDeploymentReconciler) setConditions(
	sds *deploymentv1.SingleDeploymentStatus,
	condType string,
	name string,
	message string,
	status string,
	reason string,
) {
	cond, _, found := getCondition(sds.Conditions, condType)
	if !found {
		// Add condition for deployment
		sds.Conditions = append(sds.Conditions, newCondition(
			condType,
			message,
			status,
			reason,
		))
		sds.ObservedGeneration += 1
		return
	}
	// If this field is not change, others are not changed too. Be cause they always revise together.
	backup := sds.ObservedGeneration
	if cond.Message != message {
		cond.Message = message
		backup++
	}
	if cond.Status != status {
		cond.Status = status
		backup++
	}
	if cond.Reason != reason {
		cond.Reason = reason
		backup++
	}
	if backup != sds.ObservedGeneration {
		sds.ObservedGeneration += 1
		cond.LastTransitionTime = metav1.NewTime(time.Now())
	}
}

func (r *SingleDeploymentReconciler) deleteConditions(
	sds *deploymentv1.SingleDeploymentStatus,
	condType string,
) {
	_, index, found := getCondition(sds.Conditions, condType)
	if found {
		// delete found element
		if len(sds.Conditions)-1 == index {
			// the last one, drop it
			sds.Conditions = sds.Conditions[: len(sds.Conditions)-1 : len(sds.Conditions)-1]
		} else {
			// Not the last one, swap with the last one, and drop the last one
			sds.Conditions[index], sds.Conditions[len(sds.Conditions)-1] = sds.Conditions[len(sds.Conditions)-1], sds.Conditions[index]
			sds.Conditions = sds.Conditions[: len(sds.Conditions)-1 : len(sds.Conditions)-1]
			sds.ObservedGeneration += 1
		}
	}
}

func (r *SingleDeploymentReconciler) processStatus(sds *deploymentv1.SingleDeploymentStatus) {
	isDone := true
	isFailed := false
	for i := range sds.Conditions {
		if sds.Conditions[i].Status == deploymentv1.ConditionStatusFailed {
			isFailed = true
		}
		if sds.Conditions[i].Status != deploymentv1.ConditionStatusReady {
			isDone = false
		}
	}
	if isFailed {
		r.setStatus(
			sds,
			deploymentv1.StatusPhaseFailed,
			fmt.Sprint("SingleDeployment create/update is failed"),
			deploymentv1.StatusReasonDependsUnavailable,
		)
	} else if isDone {
		r.setStatus(
			sds,
			deploymentv1.StatusPhaseSuccess,
			fmt.Sprint("SingleDeployment create/update is successed"),
			deploymentv1.StatusReasonDependsAvailable,
		)
	} else {
		r.setStatus(
			sds,
			deploymentv1.StatusPhaseRunning,
			fmt.Sprint("SingleDeployment create/update is running"),
			deploymentv1.StatusReasonDependsUnavailable,
		)
	}
}

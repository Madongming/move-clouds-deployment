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
	"reflect"
	"strings"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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

	// Watch and create/update deployment
	///////////////////////////////////////////////////////////////
	deployment := &appsv1.Deployment{}
	if err := r.Client.Get(ctx, req.NamespacedName, deployment); err != nil {
		if errors.IsNotFound(err) {
			// Its a "not found error" that is none a deployment, create it.

			// set status is creating
			sd.Status.Phase = deploymentv1.StatusCreating
			if err := r.Status().Update(ctx, sd); err != nil {
				logger.Error(err, "update status failed")
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}

			// Create deployment
			if err := r.createDeployment(ctx, logger, sd); err != nil {
				// create failed
				sd.Status.Phase = deploymentv1.StatusFailed
				if err := r.Status().Update(ctx, sd); err != nil {
					logger.Error(err, "update status failed")
					return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
				}
				return ctrl.Result{}, err
			}
		} else {
			// Its not a "not found err", throw it
			logger.Error(err, "Create deployment failed")
			sd.Status.Phase = deploymentv1.StatusFailed
			if err := r.Status().Update(ctx, sd); err != nil {
				logger.Error(err, "update status failed")
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}

			return ctrl.Result{}, err
		}
	} else {
		// Exist the deployment, update it

		// Update deployment
		if err := r.updateDeployment(ctx, logger, sd, deployment); err != nil {
			// update failed
			logger.Error(err, "Update Deployment failed")
			sd.Status.Phase = deploymentv1.StatusFailed
			if err := r.Status().Update(ctx, sd); err != nil {
				logger.Error(err, "update status failed")
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}

			return ctrl.Result{}, err
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
			sd.Status.Phase = deploymentv1.StatusCreating
			if err := r.createService(ctx, logger, sd); err != nil {
				// create failed
				logger.Error(err, "Create Service failed")
				sd.Status.Phase = deploymentv1.StatusFailed
				if err := r.Status().Update(ctx, sd); err != nil {
					logger.Error(err, "update status failed")
					return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
				}
				return ctrl.Result{}, err
			}
		} else {
			// Its not a "not found err", throw it
			logger.Error(err, "Create service failed")
			sd.Status.Phase = deploymentv1.StatusFailed
			if err := r.Status().Update(ctx, sd); err != nil {
				logger.Error(err, "update status failed")
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}
			return ctrl.Result{}, err
		}
	} else {
		// The service is exist update the service

		// Update service
		if err := r.updateService(ctx, logger, sd, service); err != nil {
			// update failed
			logger.Error(err, "update Service failed")
			sd.Status.Phase = deploymentv1.StatusFailed
			if err := r.Status().Update(ctx, sd); err != nil {
				logger.Error(err, "update status failed")
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}

			return ctrl.Result{}, err
		}
	}

	// Deal with Ingress
	ingress := new(netv1.Ingress)
	if err := r.Client.Get(ctx, req.NamespacedName, ingress); err != nil {
		if errors.IsNotFound(err) {
			if strings.ToLower(sd.Spec.Expose.Mode) == "ingress" {
				// Its a "not found error" that is none an ingress, create it.
				// Create ingress
				sd.Status.Phase = deploymentv1.StatusCreating
				if err := r.Status().Update(ctx, sd); err != nil {
					logger.Error(err, "update status failed")
					return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
				}

				if err := r.createIngress(ctx, logger, sd); err != nil {
					// create failed
					logger.Error(err, "Create Ingress failed")
					sd.Status.Phase = deploymentv1.StatusFailed
					if err := r.Status().Update(ctx, sd); err != nil {
						logger.Error(err, "update status failed")
						return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
					}
					return ctrl.Result{}, err
				}
			}
		} else {
			// Its not a "not found err", throw it
			logger.Error(err, "create ingress failed")
			sd.Status.Phase = deploymentv1.StatusFailed
			if err := r.Status().Update(ctx, sd); err != nil {
				logger.Error(err, "update status failed")
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}

			return ctrl.Result{}, err
		}
	} else {
		if strings.ToLower(sd.Spec.Expose.Mode) == "ingress" {
			// The ingress is exist and expose mode is ingress, update the ingress

			// Update ingress
			if err := r.updateIngress(ctx, logger, sd, ingress); err != nil {
				// update failed
				logger.Error(err, "update Ingress failed")
				sd.Status.Phase = deploymentv1.StatusFailed
				if err := r.Status().Update(ctx, sd); err != nil {
					logger.Error(err, "update status failed")
					return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
				}

				return ctrl.Result{}, err
			}
		} else if strings.ToLower(sd.Spec.Expose.Mode) == "nodeport" {
			// The ingress is exist, but ingressDomain is set empty, delete the ingress

			// set status is deleting
			sd.Status.Phase = deploymentv1.StatusDeleting
			if err := r.Status().Update(ctx, sd); err != nil {
				logger.Error(err, "update status failed")
				return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
			}

			// Delete ingress
			if err := r.deleteIngress(ctx, logger, ingress); err != nil {
				// update failed
				logger.Error(err, "delete Ingress failed")
				sd.Status.Phase = deploymentv1.StatusFailed
				if err := r.Status().Update(ctx, sd); err != nil {
					logger.Error(err, "update status failed")
					return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
				}

				return ctrl.Result{}, err
			}
		}
	}
	///////////////////////////////////////////////////////////////

	// All work is done
	if sd.Status.Phase != deploymentv1.StatusRunning {
		sd.Status.Phase = deploymentv1.StatusRunning
		if err := r.Status().Update(ctx, sd); err != nil {
			logger.Error(err, "update status failed")
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
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

	// set status is creating
	sd.Status.Phase = deploymentv1.StatusCreating
	if err := r.Status().Update(ctx, sd); err != nil {
		logger.Error(err, "update status failed")
		return err
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

	// set status is creating
	sd.Status.Phase = deploymentv1.StatusCreating
	if err := r.Status().Update(ctx, sd); err != nil {
		logger.Error(err, "update status failed")
		return err
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

	// set status is creating
	sd.Status.Phase = deploymentv1.StatusCreating
	if err := r.Status().Update(ctx, sd); err != nil {
		logger.Error(err, "update status failed")
		return err
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

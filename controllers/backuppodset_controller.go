/*
Copyright 2021.

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
	"encoding/hex"
	"fmt"
	"math/rand"
	"net"
	"reflect"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cri-api/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	backupv1alpha1 "github.com/destinysky/backupResourcesController/api/v1alpha1"
)

// BackupPodSetReconciler reconciles a BackupPodSet object
type BackupPodSetReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	//PodControl controller.PodControlInterface
}

// +kubebuilder:rbac:groups=backup.example.com,resources=backuppodsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=backup.example.com,resources=backuppodsets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=backup.example.com,resources=backuppodsets/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the BackupPodSet object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *BackupPodSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("backuppodset", req.NamespacedName)
	reqLogger.Info("Reconciling Backuppodset")

	// your logic here
	//selector, err := metav1.LabelSelectorAsSelector(r.Spec.Selector)
	//if err != nil {
	//	utilruntime.HandleError(fmt.Errorf("error converting pod selector to selector: %v", err))
	//	return nil
	//}
	//allPods, err := r.podLister.Pods(r.Namespace).List(labels.Everything())

	backupPodSet := &backupv1alpha1.BackupPodSet{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, backupPodSet)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	// List all replica pods owned by this instance
	lbls := labels.Set{
		"app":  backupPodSet.Name,
		"type": "primary",
	}

	existingReplicaPods := &corev1.PodList{}
	err = r.Client.List(context.TODO(),
		existingReplicaPods,
		&client.ListOptions{
			Namespace:     req.Namespace,
			LabelSelector: labels.SelectorFromSet(lbls),
		})
	if err != nil {
		reqLogger.Error(err, "failed to list existing primary pods in the backupPodSet")
		return ctrl.Result{}, err
	}
	existingPrimaryPodNames := []string{}
	// Count the primary pods that are pending or running as available
	for _, pod := range existingReplicaPods.Items {
		if pod.GetObjectMeta().GetDeletionTimestamp() != nil {
			continue
		}
		if pod.Status.Phase == corev1.PodPending || pod.Status.Phase == corev1.PodRunning {
			existingPrimaryPodNames = append(existingPrimaryPodNames, pod.GetObjectMeta().GetName())
		}
	}

	// List all hotbackup pods owned by this instance
	lbls = labels.Set{
		"app":  backupPodSet.Name,
		"type": "hotbackup",
	}
	existingHotbackupPods := &corev1.PodList{}
	err = r.Client.List(context.TODO(),
		existingHotbackupPods,
		&client.ListOptions{
			Namespace:     req.Namespace,
			LabelSelector: labels.SelectorFromSet(lbls),
		})
	if err != nil {
		reqLogger.Error(err, "failed to list existing hotbackup pods in the backupPodSet")
		return ctrl.Result{}, err
	}
	existingHotbackupPodNames := []string{}
	// Count the hotbackup pods that are pending or running as available
	for _, pod := range existingHotbackupPods.Items {
		if pod.GetObjectMeta().GetDeletionTimestamp() != nil {
			continue
		}
		if pod.Status.Phase == corev1.PodPending || pod.Status.Phase == corev1.PodRunning {
			existingHotbackupPodNames = append(existingHotbackupPodNames, pod.GetObjectMeta().GetName())
		}
	}

	// List all coldbackup pods owned by this instance
	lbls = labels.Set{
		"app":  backupPodSet.Name,
		"type": "coldbackup",
	}
	existingColdbackupPods := &corev1.PodList{}
	err = r.Client.List(context.TODO(),
		existingColdbackupPods,
		&client.ListOptions{
			Namespace:     req.Namespace,
			LabelSelector: labels.SelectorFromSet(lbls),
		})
	if err != nil {
		reqLogger.Error(err, "failed to list existing coldbackup pods in the backupPodSet")
		return ctrl.Result{}, err
	}
	existingColdbackupPodNames := []string{}
	// Count the coldbackup pods that are pending or running as available
	for _, pod := range existingColdbackupPods.Items {
		if pod.GetObjectMeta().GetDeletionTimestamp() != nil {
			continue
		}
		if pod.Status.Phase == corev1.PodPending || pod.Status.Phase == corev1.PodRunning {
			existingColdbackupPodNames = append(existingColdbackupPodNames, pod.GetObjectMeta().GetName())
		}
	}

	// Update the status if necessary
	status := backupv1alpha1.BackupPodSetStatus{
		Replicas:           int32(len(existingPrimaryPodNames)),
		PrimaryPodNames:    existingPrimaryPodNames,
		HotBackups:         int32(len(existingHotbackupPodNames)),
		HotBackupPodNames:  existingHotbackupPodNames,
		ColdBackups:        int32(len(existingColdbackupPodNames)),
		ColdBackupPodNames: existingColdbackupPodNames,
	}
	if !reflect.DeepEqual(backupPodSet.Status, status) {
		backupPodSet.Status = status
		err := r.Client.Status().Update(context.TODO(), backupPodSet)
		if err != nil {
			reqLogger.Error(err, "failed to update the BackupPodSetStatus")
			return ctrl.Result{}, err
		}
	}
	annotations := backupPodSet.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations["backup.example.com/replicas"] = strconv.Itoa((int(len(existingPrimaryPodNames)))) + "/" + strconv.Itoa(int(backupPodSet.Spec.Replicas))
	annotations["backup.example.com/hotbackups"] = strconv.Itoa((int(len(existingHotbackupPodNames)))) + "/" + strconv.Itoa(int(backupPodSet.Spec.HotBackups))
	annotations["backup.example.com/coldbackups"] = strconv.Itoa((int(len(existingColdbackupPodNames)))) + "/" + strconv.Itoa(int(backupPodSet.Spec.ColdBackups))
	//if !reflect.DeepEqual(backupPodSet.GetAnnotations(), annotations) {
	reqLogger.Info("Changing Annotations")
	backupPodSet.SetAnnotations(annotations)
	err = r.Client.Update(context.TODO(), backupPodSet)
	if err != nil {
		reqLogger.Error(err, "failed to update the BackupPodSetStatus annotations")
		return ctrl.Result{}, err
	}
	//}

	// Init when create a new resource

	if int32(len(existingPrimaryPodNames)) == 0 && int32(len(existingHotbackupPodNames)) == 0 && int32(len(existingColdbackupPodNames)) == 0 {
		reqLogger.Info("Initialize the backupPodSet, name " + backupPodSet.Name)
		//if backupPodSet.Spec.Strategy.Init == "random" {

		//}
		//else{
		//	reqLogger.Error("failed to initialize the backupPodSet, name" , backupPodSet.Name , ". Unknow strategy" , backupPodSet.Spec.Strategy.Init)
		//	return ctrl.Result{}, fmt.Errorf("failed to initialize the backupPodSet, name %v. Unknow strategy %v",backupPodSet.Name, backupPodSet.Spec.Strategy.Init)
		//}
		for i := int32(0); i < backupPodSet.Spec.Replicas; i++ {
			reqLogger.Info("Adding a primary pod in the backupPodSet in init phase" + backupPodSet.Name)
			pod, err := newPod(backupPodSet, "primary", "init", backupPodSet.Spec.SchedulerName)
			if err != nil {
				reqLogger.Error(err, "failed to init the BackupPodSet for primary")
				return ctrl.Result{}, err
			}
			if err = controllerutil.SetControllerReference(backupPodSet, pod, r.Scheme); err != nil {
				reqLogger.Error(err, "unable to set owner reference on new pod")
				return ctrl.Result{}, err
			}
			err = r.Client.Create(context.TODO(), pod)
			if err != nil {
				reqLogger.Error(err, "failed to create a pod")
				return ctrl.Result{}, err
			}

		}

		for i := int32(0); i < backupPodSet.Spec.HotBackups; i++ {
			reqLogger.Info("Adding a hotbackup pod in the backupPodSet in init phase" + backupPodSet.Name)
			pod, err := newPod(backupPodSet, "hotbackup", "init", backupPodSet.Spec.SchedulerName)
			if err != nil {
				reqLogger.Error(err, "failed to init the BackupPodSet for hotbackup")
				return ctrl.Result{}, err
			}
			if err = controllerutil.SetControllerReference(backupPodSet, pod, r.Scheme); err != nil {
				reqLogger.Error(err, "unable to set owner reference on new pod")
				return ctrl.Result{}, err
			}
			err = r.Client.Create(context.TODO(), pod)
			if err != nil {
				reqLogger.Error(err, "failed to create a pod")
				return ctrl.Result{}, err
			}

		}

		for i := int32(0); i < backupPodSet.Spec.ColdBackups; i++ {
			reqLogger.Info("Adding a coldbackup pod in the backupPodSet in init phase" + backupPodSet.Name)
			pod, err := newPod(backupPodSet, "coldbackup", "init", backupPodSet.Spec.SchedulerName)
			if err != nil {
				reqLogger.Error(err, "failed to init the BackupPodSet for cold backup")
				return ctrl.Result{}, err
			}
			if err = controllerutil.SetControllerReference(backupPodSet, pod, r.Scheme); err != nil {
				reqLogger.Error(err, "unable to set owner reference on new pod")
				return ctrl.Result{}, err
			}
			err = r.Client.Create(context.TODO(), pod)
			if err != nil {
				reqLogger.Error(err, "failed to create a pod")
				return ctrl.Result{}, err
			}

		}
		_, err := updateStatus(r, req, reqLogger)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil

	} else {

		//Not init
		if int32(len(existingPrimaryPodNames)) < backupPodSet.Spec.Replicas {
			reqLogger.Info("Existing primary pods " + strconv.Itoa((int(len(existingPrimaryPodNames)))) + " is less than number of replicas in yaml " + strconv.Itoa(int(backupPodSet.Spec.Replicas)))
			//HB to P
			for {
				if int32(len(existingPrimaryPodNames)) == backupPodSet.Spec.Replicas || int32(len(existingHotbackupPodNames)) == 0 {
					break
				}
				reqLogger.Info("Converting a hotbakcup to primary" + backupPodSet.Name)
				pod := &v1.Pod{}
				err := r.Client.Get(context.TODO(), types.NamespacedName{
					Namespace: req.Namespace,
					Name:      existingHotbackupPodNames[0],
				}, pod)
				if err != nil {
					// Error reading the object - requeue the request.
					return ctrl.Result{}, err
				}
				pod.Labels["type"] = "primary"
				err = r.Client.Update(context.TODO(), pod)
				if err != nil {
					// Error reading the object - requeue the request.
					return ctrl.Result{}, err
				}

				existingReplicaPods.Items = append(existingReplicaPods.Items, existingHotbackupPods.Items[0])
				existingHotbackupPods.Items = existingHotbackupPods.Items[1:]
				existingPrimaryPodNames = append(existingPrimaryPodNames, existingHotbackupPodNames[0])
				existingHotbackupPodNames = existingHotbackupPodNames[1:]

			}

			//CB to P
			for {
				if int32(len(existingPrimaryPodNames)) == backupPodSet.Spec.Replicas || int32(len(existingColdbackupPodNames)) == 0 {
					break
				}
				reqLogger.Info("Converting a coldbakcup to primary" + backupPodSet.Name)
				pod := &v1.Pod{}
				err := r.Client.Get(context.TODO(), types.NamespacedName{
					Namespace: req.Namespace,
					Name:      existingColdbackupPodNames[0],
				}, pod)
				if err != nil {
					// Error reading the object - requeue the request.
					return ctrl.Result{}, err
				}

				ip := pod.Status.PodIP

				if ip != "" {
					pod.Labels["type"] = "primary"
					ok, err := wakeupColdbackup(ip, backupPodSet.Spec.WakeupTimeout)
					if err != nil {
						// Error reading the object - requeue the request.
						return ctrl.Result{}, err
					}
					if !ok {
						err = r.Client.Delete(context.TODO(), pod)
						if err != nil {
							reqLogger.Error(err, "failed to delete a pod")
							return ctrl.Result{}, err
						}
						break
					}
					err = r.Client.Update(context.TODO(), pod)
					if err != nil {
						// Error reading the object - requeue the request.
						return ctrl.Result{}, err
					}
					existingReplicaPods.Items = append(existingReplicaPods.Items, existingColdbackupPods.Items[0])
					existingPrimaryPodNames = append(existingPrimaryPodNames, existingColdbackupPodNames[0])
				} else {
					err = r.Client.Delete(context.TODO(), pod)
					if err != nil {
						reqLogger.Error(err, "failed to delete a pod")
						return ctrl.Result{}, err
					}
				}
				existingColdbackupPods.Items = existingColdbackupPods.Items[1:]
				existingColdbackupPodNames = existingColdbackupPodNames[1:]
			}

			//U to P
			for {
				if int32(len(existingPrimaryPodNames)) == backupPodSet.Spec.Replicas {
					break
				}
				reqLogger.Info("Converting an unallocated resource to primary" + backupPodSet.Name)
				pod, err := newPod(backupPodSet, "primary", "toprimary", backupPodSet.Spec.SchedulerName)
				if err != nil {
					reqLogger.Error(err, "failed to add new primary pods from unallocated pods")
					return ctrl.Result{}, err
				}
				if err = controllerutil.SetControllerReference(backupPodSet, pod, r.Scheme); err != nil {
					reqLogger.Error(err, "unable to set owner reference on new pod")
					return ctrl.Result{}, err
				}
				err = r.Client.Create(context.TODO(), pod)
				if err != nil {
					reqLogger.Error(err, "failed to create a pod")
					return ctrl.Result{}, err
				}
				existingReplicaPods.Items = append(existingReplicaPods.Items, *pod)
				existingPrimaryPodNames = append(existingPrimaryPodNames, pod.Name)
			}

		} else if int32(len(existingPrimaryPodNames)) > backupPodSet.Spec.Replicas {
			reqLogger.Info("Deleting " + strconv.Itoa(int(len(existingPrimaryPodNames))-int(backupPodSet.Spec.Replicas)) + "primary pod(s) in the backupPodSet")
			for i := int32(0); i < int32(len(existingPrimaryPodNames))-backupPodSet.Spec.Replicas; i++ {
				pod := existingReplicaPods.Items[0]
				err = r.Client.Delete(context.TODO(), &pod)
				if err != nil {
					reqLogger.Error(err, "failed to delete a pod")
					return ctrl.Result{}, err
				}
				existingReplicaPods.Items = existingReplicaPods.Items[1:]
				existingPrimaryPodNames = existingPrimaryPodNames[1:]
			}
		}

		if int32(len(existingPrimaryPodNames)) == backupPodSet.Spec.Replicas {
			reqLogger.Info("Existing primary pods " + strconv.Itoa(int(len(existingPrimaryPodNames))) + " is equal to number of replicas in yaml " + strconv.Itoa(int(backupPodSet.Spec.Replicas)))
			if int32(len(existingHotbackupPodNames)) < backupPodSet.Spec.HotBackups {
				//C to H
				reqLogger.Info("Existing hotbackup pods " + strconv.Itoa(int(len(existingHotbackupPodNames))) + " is less than number of hotbackup pods in yaml " + strconv.Itoa(int(backupPodSet.Spec.HotBackups)))
				for {
					if int32(len(existingHotbackupPodNames)) == backupPodSet.Spec.HotBackups || int32(len(existingColdbackupPodNames)) == 0 {
						break
					}
					reqLogger.Info("Converting an coldbackup resource to hotbackup" + backupPodSet.Name)
					pod := &v1.Pod{}
					err := r.Client.Get(context.TODO(), types.NamespacedName{
						Namespace: req.Namespace,
						Name:      existingColdbackupPodNames[0],
					}, pod)
					if err != nil {
						// Error reading the object - requeue the request.
						return ctrl.Result{}, err
					}

					ip := pod.Status.PodIP

					if ip != "" {
						pod.Labels["type"] = "hotbackup"
						ok, err := wakeupColdbackup(ip, backupPodSet.Spec.WakeupTimeout)
						if err != nil {
							// Error reading the object - requeue the request.
							return ctrl.Result{}, err
						}
						if !ok {
							err = r.Client.Delete(context.TODO(), pod)
							if err != nil {
								reqLogger.Error(err, "failed to delete a pod")
								return ctrl.Result{}, err
							}
							break
						}
						err = r.Client.Update(context.TODO(), pod)
						if err != nil {
							// Error reading the object - requeue the request.
							return ctrl.Result{}, err
						}
						existingHotbackupPods.Items = append(existingHotbackupPods.Items, existingColdbackupPods.Items[0])
						existingHotbackupPodNames = append(existingHotbackupPodNames, existingColdbackupPodNames[0])
					} else {
						err = r.Client.Delete(context.TODO(), pod)
						if err != nil {
							reqLogger.Error(err, "failed to delete a pod")
							return ctrl.Result{}, err
						}
					}
					existingColdbackupPods.Items = existingColdbackupPods.Items[1:]
					existingColdbackupPodNames = existingColdbackupPodNames[1:]
				}

				//U to H

				for {
					if int32(len(existingHotbackupPodNames)) == backupPodSet.Spec.HotBackups {
						break
					}
					reqLogger.Info("Converting an unallocated resource to hotbackup" + backupPodSet.Name)
					pod, err := newPod(backupPodSet, "hotbackup", "tohot", backupPodSet.Spec.SchedulerName)
					if err != nil {
						reqLogger.Error(err, "failed to add new hotbackup pods from unallocated pods")
						return ctrl.Result{}, err
					}
					if err = controllerutil.SetControllerReference(backupPodSet, pod, r.Scheme); err != nil {
						reqLogger.Error(err, "unable to set owner reference on new pod")
						return ctrl.Result{}, err
					}
					err = r.Client.Create(context.TODO(), pod)
					if err != nil {
						reqLogger.Error(err, "failed to create a pod")
						return ctrl.Result{}, err
					}
					existingHotbackupPods.Items = append(existingHotbackupPods.Items, *pod)
					existingHotbackupPodNames = append(existingHotbackupPodNames, pod.Name)
				}

			} else if int32(len(existingHotbackupPodNames)) > backupPodSet.Spec.HotBackups {
				reqLogger.Info("Deleting " + strconv.Itoa(int(len(existingHotbackupPodNames))-int(backupPodSet.Spec.HotBackups)) + "hotbackup pod(s) in the backupPodSet")
				for i := int32(0); i < int32(len(existingHotbackupPodNames))-backupPodSet.Spec.HotBackups; i++ {
					pod := existingHotbackupPods.Items[0]
					err = r.Client.Delete(context.TODO(), &pod)
					if err != nil {
						reqLogger.Error(err, "failed to delete a pod")
						return ctrl.Result{}, err
					}
					existingHotbackupPods.Items = existingHotbackupPods.Items[1:]
					existingHotbackupPodNames = existingHotbackupPodNames[1:]
				}

			}

			if int32(len(existingHotbackupPodNames)) == backupPodSet.Spec.HotBackups {
				reqLogger.Info("Existing hotbackup pods " + strconv.Itoa(int(len(existingPrimaryPodNames))) + " is equal to number of hotbackup in yaml " + strconv.Itoa(int(backupPodSet.Spec.Replicas)))
				if int32(len(existingColdbackupPodNames)) < backupPodSet.Spec.ColdBackups {
					reqLogger.Info("Existing hotbackup pods " + strconv.Itoa(int(len(existingPrimaryPodNames))) + " is less than number of hotbackup in yaml " + strconv.Itoa(int(backupPodSet.Spec.Replicas)))
					//U to C
					for {
						if int32(len(existingColdbackupPodNames)) == backupPodSet.Spec.ColdBackups {
							break
						}
						reqLogger.Info("Converting an unallocated resource to coldbackup" + backupPodSet.Name)
						pod, err := newPod(backupPodSet, "coldbackup", "tocold", backupPodSet.Spec.SchedulerName)
						if err != nil {
							reqLogger.Error(err, "failed to add new coldbackup pods from unallocated pods")
							return ctrl.Result{}, err
						}
						if err = controllerutil.SetControllerReference(backupPodSet, pod, r.Scheme); err != nil {
							reqLogger.Error(err, "unable to set owner reference on new pod")
							return ctrl.Result{}, err
						}
						err = r.Client.Create(context.TODO(), pod)
						if err != nil {
							reqLogger.Error(err, "failed to create a pod")
							return ctrl.Result{}, err
						}
						existingColdbackupPods.Items = append(existingColdbackupPods.Items, *pod)
						existingColdbackupPodNames = append(existingColdbackupPodNames, pod.Name)
					}

				} else if int32(len(existingColdbackupPodNames)) > backupPodSet.Spec.ColdBackups {
					reqLogger.Info("Deleting " + strconv.Itoa(int(len(existingColdbackupPodNames))-int(backupPodSet.Spec.ColdBackups)) + "coldbackup pod(s) in the backupPodSet")
					for i := int32(0); i < int32(len(existingColdbackupPodNames))-backupPodSet.Spec.ColdBackups; i++ {
						pod := existingColdbackupPods.Items[0]
						err = r.Client.Delete(context.TODO(), &pod)
						if err != nil {
							reqLogger.Error(err, "failed to delete a pod")
							return ctrl.Result{}, err
						}
						existingColdbackupPods.Items = existingColdbackupPods.Items[1:]
						existingColdbackupPodNames = existingColdbackupPodNames[1:]
					}
				}

				if int32(len(existingColdbackupPodNames)) == backupPodSet.Spec.ColdBackups {
					reqLogger.Info("Existing coldbackup pods " + strconv.Itoa(int(len(existingPrimaryPodNames))) + " is equal to number of coldbackup in yaml " + strconv.Itoa(int(backupPodSet.Spec.Replicas)))
					_, err := updateStatus(r, req, reqLogger)
					if err != nil {
						return ctrl.Result{}, err
					}
					return ctrl.Result{}, nil
				}
			}
		}
	}
	return ctrl.Result{Requeue: true}, nil
}

// newPod returns a pod with the same name/namespace as the cr
func newPod(cr *backupv1alpha1.BackupPodSet, tp string, event string, schedulername string) (*corev1.Pod, error) {
	rand.Seed(time.Now().UnixNano())
	randBytes := make([]byte, 12/2)
	rand.Read(randBytes)

	templete := cr.Spec.Template

	var podname string

	if templete.GetName() != "" {
		podname = templete.GetName() + "-" + cr.Name + "-" + hex.EncodeToString(randBytes)
	} else {
		podname = cr.Name + "-" + hex.EncodeToString(randBytes)
	}

	templete.SetName(podname)
	templete.SetNamespace(cr.Namespace)
	labels := templete.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	labels["app"] = cr.Name
	if event == "init" {
		labels["strategy"] = cr.Spec.Strategy.Init
	} else if event == "tocold" {
		labels["strategy"] = cr.Spec.Strategy.UnallocatedToColdbackup
	} else if event == "tohot" {
		labels["strategy"] = cr.Spec.Strategy.UnallocatedToHotbackup
	} else if event == "toprimary" {
		labels["strategy"] = cr.Spec.Strategy.UnallocatedToReplicas
	} else {
		return nil, fmt.Errorf("Unknow strategy %s", event)
	}

	if tp == "primary" || tp == "hotbackup" {
		labels["type"] = tp
	} else if tp == "coldbackup" {
		labels["type"] = tp
		templete.Spec.InitContainers = append(templete.Spec.InitContainers, corev1.Container{
			Image: "destinysky/coldbackup:0.1",
			Name:  "coldbackup",
		})
	} else {
		return nil, fmt.Errorf("Unknow Pod type %s", tp)
	}

	templete.SetLabels(labels)

	if schedulername == "" {
		schedulername = "default-scheduler"
	}
	templete.Spec.SchedulerName = schedulername

	return &corev1.Pod{
		ObjectMeta: templete.ObjectMeta,
		Spec:       templete.Spec,
	}, nil

	//return &corev1.Pod{templete}, nil
}

func wakeupColdbackup(ip string, timeout int32) (bool, error) {
	conn, err := net.Dial("tcp", ip+":20000")
	defer conn.Close()
	if err != nil {
		return false, err
	}

	quit := make(chan int)
	ok := false

	go func(quit <-chan int) {
		for {
			select {
			case <-quit:
				return
			default:
			}
			conn.Write([]byte("ok"))
			//time.Sleep(time.Duration(1) * time.Second)
		}

	}(quit)

	for i := int32(0); i < timeout; i++ {

		buffer := make([]byte, 1024)
		n, err := conn.Read(buffer)
		if err != nil {
			return false, err
		}

		serverMsg := string(buffer[0:n])
		if serverMsg == "bye" {
			ok = true
			break
		}
	}
	quit <- 1

	return ok, nil
}

func updateStatus(r *BackupPodSetReconciler, req ctrl.Request, reqLogger logr.Logger) (bool, error) {
	backupPodSet := &backupv1alpha1.BackupPodSet{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, backupPodSet)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return false, nil
		}
		// Error reading the object - requeue the request.
		return false, err
	}

	// List all replica pods owned by this instance
	lbls := labels.Set{
		"app":  backupPodSet.Name,
		"type": "primary",
	}

	existingReplicaPods := &corev1.PodList{}
	err = r.Client.List(context.TODO(),
		existingReplicaPods,
		&client.ListOptions{
			Namespace:     req.Namespace,
			LabelSelector: labels.SelectorFromSet(lbls),
		})
	if err != nil {
		reqLogger.Error(err, "failed to list existing primary pods in the backupPodSet")
		return false, err
	}
	existingPrimaryPodNames := []string{}
	// Count the primary pods that are pending or running as available
	for _, pod := range existingReplicaPods.Items {
		if pod.GetObjectMeta().GetDeletionTimestamp() != nil {
			continue
		}
		if pod.Status.Phase == corev1.PodPending || pod.Status.Phase == corev1.PodRunning {
			existingPrimaryPodNames = append(existingPrimaryPodNames, pod.GetObjectMeta().GetName())
		}
	}

	// List all hotbackup pods owned by this instance
	lbls = labels.Set{
		"app":  backupPodSet.Name,
		"type": "hotbackup",
	}
	existingHotbackupPods := &corev1.PodList{}
	err = r.Client.List(context.TODO(),
		existingHotbackupPods,
		&client.ListOptions{
			Namespace:     req.Namespace,
			LabelSelector: labels.SelectorFromSet(lbls),
		})
	if err != nil {
		reqLogger.Error(err, "failed to list existing hotbackup pods in the backupPodSet")
		return false, err
	}
	existingHotbackupPodNames := []string{}
	// Count the hotbackup pods that are pending or running as available
	for _, pod := range existingHotbackupPods.Items {
		if pod.GetObjectMeta().GetDeletionTimestamp() != nil {
			continue
		}
		if pod.Status.Phase == corev1.PodPending || pod.Status.Phase == corev1.PodRunning {
			existingHotbackupPodNames = append(existingHotbackupPodNames, pod.GetObjectMeta().GetName())
		}
	}

	// List all coldbackup pods owned by this instance
	lbls = labels.Set{
		"app":  backupPodSet.Name,
		"type": "coldbackup",
	}
	existingColdbackupPods := &corev1.PodList{}
	err = r.Client.List(context.TODO(),
		existingColdbackupPods,
		&client.ListOptions{
			Namespace:     req.Namespace,
			LabelSelector: labels.SelectorFromSet(lbls),
		})
	if err != nil {
		reqLogger.Error(err, "failed to list existing coldbackup pods in the backupPodSet")
		return false, err
	}
	existingColdbackupPodNames := []string{}
	// Count the coldbackup pods that are pending or running as available
	for _, pod := range existingColdbackupPods.Items {
		if pod.GetObjectMeta().GetDeletionTimestamp() != nil {
			continue
		}
		if pod.Status.Phase == corev1.PodPending || pod.Status.Phase == corev1.PodRunning {
			existingColdbackupPodNames = append(existingColdbackupPodNames, pod.GetObjectMeta().GetName())
		}
	}

	// Update the status if necessary
	status := backupv1alpha1.BackupPodSetStatus{
		Replicas:           int32(len(existingPrimaryPodNames)),
		PrimaryPodNames:    existingPrimaryPodNames,
		HotBackups:         int32(len(existingHotbackupPodNames)),
		HotBackupPodNames:  existingHotbackupPodNames,
		ColdBackups:        int32(len(existingColdbackupPodNames)),
		ColdBackupPodNames: existingColdbackupPodNames,
	}
	if !reflect.DeepEqual(backupPodSet.Status, status) {
		backupPodSet.Status = status
		err := r.Client.Status().Update(context.TODO(), backupPodSet)
		if err != nil {
			reqLogger.Error(err, "failed to update the BackupPodSetStatus")
			return false, err
		}
	}
	annotations := backupPodSet.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations["backup.example.com/replicas"] = strconv.Itoa((int(len(existingPrimaryPodNames)))) + "/" + strconv.Itoa(int(backupPodSet.Spec.Replicas))
	annotations["backup.example.com/hotbackups"] = strconv.Itoa((int(len(existingHotbackupPodNames)))) + "/" + strconv.Itoa(int(backupPodSet.Spec.HotBackups))
	annotations["backup.example.com/coldbackups"] = strconv.Itoa((int(len(existingColdbackupPodNames)))) + "/" + strconv.Itoa(int(backupPodSet.Spec.ColdBackups))
	//if !reflect.DeepEqual(backupPodSet.GetAnnotations(), annotations) {
	reqLogger.Info("Changing Annotations")
	backupPodSet.SetAnnotations(annotations)
	err = r.Client.Update(context.TODO(), backupPodSet)
	if err != nil {
		reqLogger.Error(err, "failed to update the BackupPodSetStatus annotations")
		return false, err
	}
	//}
	return true, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BackupPodSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&backupv1alpha1.BackupPodSet{}).
		Complete(r)
}

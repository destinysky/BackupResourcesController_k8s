# BackupResourcesController_k8s

This application designs and implements a custom resource and the corresponding controller in Kubernetes to manage the primary and backup resources of network functions. The custom resource is a set of Pods with different types, which includes primary, hot backup, and cold backup Pods. The controller manages the set of Pods and maintains the current state of the different types of Pods to keep the current state consistent with the desired state of each type of Pod.

## Motivations and goals

- Motivations:  
  - Backup resource management plays a key role in network function virtualization since a network function may fail due to multiple reasons.
  - The costs for resource management and maintenance account for a large part of the entire life cycle of a network software.
  - However, Kubernetes does not provide a resource type to define the backup Pods with considering the different backup strategies. In addition, it does not provide automatic resource management based on user requests for the backup Pods.

- Goals
  - Design and implement a controller to manage the primary and backup resources of network functions
    - with introducing a new resource type called backup Pod set (BPS), which is a custom resource in Kubernetes

## Structure

![Structure figure 1](https://github.com/destinysky/resources/raw/master/BackupResourcesController_k8s/structure.png)

- Object: BPS includes a certain number of different types of Pods which are the primary, HB, and CB Pods.
  - The backup resource protected with the CB strategy is only reserved without being activated;
  - the backup resource protected with the HB strategy is activated and synchronized with the primary resource before any failure occurs.
- When a BPS instance is requested to be created, updated, or deleted,
  - Informer, which is a bridge between the API server and the controller, receives the notification from the API server
  - it pushes the events caused by BPS instance operations to a First-In, First-Out (FIFO) queue called WorkQueue.

![Control loop figure 1](https://github.com/destinysky/resources/raw/master/BackupResourcesController_k8s/state2.png)

- Control loop is the core of the controller, which handles the events in WorkQueue by maintaining the current state of a BPS instance until it is consistent with the desired state.
  - Control loop maintains the number of Pods for each type by resource releasing and converting, until the number of Pods for each type is consistent with the desired number requested by the user requirement
  - The scheduler decides the allocations of the newly created Pods with the default  scheduler or allocation-model-based scheduler in [R. Kang 2021].

    [R. Kang 2021] R. Kang, etc., “Design of scheduler plugins for reliable function allocation in kubernetes,” in Conf. Design of Reliable Commun. Netw. (DRCN2021).

## BPS instance

- A BPS instance is a custom resource defined by CRD.
  - CRD includes the required information and specifications for creating a BPS instance with three parts: Metadata, Spec, and Status.
  - Configuration file of a BPS instance:
    ![BPS instance figure 1](https://github.com/destinysky/resources/raw/master/BackupResourcesController_k8s/yaml4.png)
- Primary, HB, and CB Pods are distinguished by the labels
  - Each Pod with a primary label is used to balance the traffic for services.
  - Each Pod with a hot backup label is activated without being exposed to the service; the Pod takes over the tasks of primary Pod once a failure is detected.
  - When a Pod with a cold backup label is requested to be created, we add an init container in the CB Pod. Init containers run before the other containers in the same Pod. It waits for the activation message from the control loop.

## Control loop

![Algorithm figure 1](https://github.com/destinysky/resources/raw/master/BackupResourcesController_k8s/algorithm1.png)

- Firstly, the current HB Pod is converted to the primary Pod and exposed to the service.
- Secondly, the current CB Pod is converted to the primary Pod with being activated and exposed to the service.
- Thirdly, the remaining primary Pods are created with using unallocated resources.
- When the current number of the primary/CB/HB Pods is larger than the desired number, each unnecessary Pod is deleted; the occupied resources of the unnecessary Pods are released.

## Demonstration

- Software:
  - Controller by Operator SDK v1.4.2 and Golang 1.15.
  - Data collector by Python 3.9.
  - Images built by Docker 20.10.0.
  - The demonstrations run on a four-node Kubernetes cluster, one master node, and three worker nodes. The version of Kubernetes is  v1.20.4.
  - Runing on an Intel Core i7-10510U 1.80 GHz 2-core CPU, 4 GB memory.

    ![Demo figure 1](https://github.com/destinysky/resources/raw/master/BackupResourcesController_k8s/controllerPod.png)
- BPS creation:
  - Create a BPS instance with two primary Pods, two HB Pods, and two CB Pods by a YAML file
  - The time for the controller to handle the BPS creation request is 0.211 [s]. The total creation time for the instance is 5.344 [s].

    ![Demo figure 2](https://github.com/destinysky/resources/raw/master/BackupResourcesController_k8s/deploy.png)

- When the traffic increases, the request of updating the number of Pods for each type in a BPS instance from two to three for load balancing arrives.
  - The controller compares the current and desired states of the different types of Pods; one Pod for each type is added.
  - One HB Pod 3, is converted to a primary Pod.
  - Two CB Pods (5, 6) are activated and converted to HB Pods.
  - Three Pods (7, 8, 9) are newly created as the CB Pods.
  - The time for the controller to handle the BPS updating request is 0.113 [s]. The total transition time from the last state to the current one of the BPS instance is 3.681 [s].

    ![Demo figure 3](https://github.com/destinysky/resources/raw/master/BackupResourcesController_k8s/edittable.png)
    ![Demo figure 4](https://github.com/destinysky/resources/raw/master/BackupResourcesController_k8s/edit.png)

- In the case that a primary Pod failure is detected, the HB and CB Pods are activated to take over the task of the failed primary Pod.
  - The HB Pod, Pod 6, is converted to a primary Pod and is exposed to the service.
  - The CB Pod, Pod 9, is successfully activated and converted to an HB Pod.
  - A CB Pod, Pod 10, is newly created.
  - The time for the controller to handle a primary Pod failure is 0.079 [s]. The total transition time is 1.933 [s].

    ![Demo figure 5](https://github.com/destinysky/resources/raw/master/BackupResourcesController_k8s/deltable.png)
    ![Demo figure 6](https://github.com/destinysky/resources/raw/master/BackupResourcesController_k8s/del.png)

## Citation

If these codes are helpful to your work, please cite this paper. Thank you.

>M. Zhu, R. Kang, F. He, and E. Oki, “Implementation of backup resource
management controller for reliable function allocation in kubernetes,”
in 2021 IEEE 7th International Conference on Network Softwarization
(NetSoft) (NetSoft 2021), Tokyo, Japan, Jun. 2021.

>@INPROCEEDINGS{xxxx,
AUTHOR="Mengfei Zhu and Rui Kang and Fujun He and Eiji Oki",
TITLE="Implementation of Backup Resource Management Controller for Reliable
Function Allocation in Kubernetes",
BOOKTITLE="2021 IEEE 7th International Conference on Network Softwarization (NetSoft)
(NetSoft 2021)",
ADDRESS="Tokyo, Japan",
DAYS=27,
MONTH=jun,
YEAR=2021
}

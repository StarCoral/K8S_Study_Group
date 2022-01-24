# Assignment - Scheduling Framework
#### tags: `Assignment`, `Scheduling`
- **Update Date:** Jan 24, 2022
- **Author:** StarCoral

## Requirement

- **Environment:** kind 1.17.17
- **Language:** go1.17.6


## Problem Description

Currently, default kubernetes scheduler can only do FIFO(First-In-First-Out) scheduling.  
In this assignment, we are going to implement a custom scheduler by [Scheduling framework](https://kubernetes.io/docs/concepts/scheduling-eviction/scheduling-framework/).  
It schedules jobs(pods) according to the following rules:  
1. Workloads will be assigned the label(`groupName`) by user, which is used to distinguish the different groups.
2. User will also assigned the label(`groupPriority`) to determine the scheduling priority of different groups.
3. The priority of every workload in same group depends on [QoS level](https://kubernetes.io/docs/tasks/configure-pod-container/quality-service-pod/) (`Guaranteed`>`Burstable`>`BestEffort`) by user.
4. Each group will start to schedule only when the number of pods in the group >= the minimum available number(`minAvailable`)

### Label Description
Yaml file will provide the following labels, which can be used for judgement.  
Please note that when testing,   
we will assume that pods with the same podGroup settings will have same `miniAvailable` and `groupPriority`   

+ `groupName`: the group name of the workload.
    + If the workload has same name, they will be viewed as same group. Otherwise, they are different groups.
+ `groupPriority`: the priority of the group.
    + The higher groupPriority value, the higher scheduling priority.
+ `minAvailable`: the minimum available number of group.

## Hint  

+ There are three parts you need to do:
    1. Trace the code
    2. Tag “TODO” is the place you need to modify or implement
    3. Deploy your own program & Test
+ Implement plugin：
    + **QueueSort**
        +  Compare the priority of pods first.
        +  If the priority of pods are equal, then comapre the QOS 
    + **PreFilter**
        + If the number of pods in same group < minAvailable,  then the pod will reject and retry later.
+ **Writing logs** is a good habit.

## Check

+ Check your program is working.
```bash=
$ kubectl get pod -n kube-system
```

+ Input Example1

    + Description:
    ```
        There is a job, `nginx` and its group is also called the same name.
        According the file of yaml, we can see there is a constraint that this service must has 3 pods, 
        then the pods will start to be scheduled to the nodes.
        
        Actually, the operator made a mistake, he accidentally typed the wrong number.

        Thus, we can see ...

        The number of Pods in same group < minAvailable, so the pod status is `Pending`.

        When we check the log, we can find the situation that the number of pods in same group is less than minAvailable.
    ```
    + Command:
    ```bash=
        $ kubectl apply -f {filename}
    ```
    + Test file:
    ```yaml=
    apiVersion: batch/v1
    kind: Job
    metadata:
    name: nginx
    spec:
    parallelism: 2
    completions: 2
    template:
        metadata:
        labels:
            app: nginx
            podGroup: nginx
            minAvailable: "3"
        spec:
        schedulerName: riya-scheduler
        containers:
            - name: nginx
            image:  nginx
            resources:
                requests:
                cpu: 3000m
                memory: 500Mi
                limits:
                cpu: 3000m
                memory: 500Mi
        restartPolicy: Never
    ```
+ Output Example 1
    + Command:
    ```bash=
    $ kubectl get pod
    ```
    ![](https://i.imgur.com/AuqzFXZ.png)
    + log:
    ```bash=
    $ kubectl logs {your scheduler pod name} -n kube-system
    ```
    ![](https://i.imgur.com/rq36Gje.png)

![](https://i.imgur.com/2SdDCGG.png)


## Reference
+ Schduling framework：
https://github.com/kubernetes/enhancements/blob/master/keps/sig-scheduling/20180409-scheduling-framework.md
+ k8s testing：   https://www.reddit.com/r/kubernetes/comments/be0415/k3s_minikube_or_microk8s/
+ kubernetes in docker(kind)：
  https://kind.sigs.k8s.io/docs/user/quick-start/
+ Go language：
  https://golang.org/
+ Dockerfile：
  https://docs.docker.com/engine/reference/builder/
+ client-go
  https://github.com/kubernetes/client-go
+ k8s qos
  https://kubernetes.io/docs/tasks/configure-pod-container/quality-service-pod/
+ sample-scheduler-framework code
  https://github.com/cnych/sample-scheduler-framework


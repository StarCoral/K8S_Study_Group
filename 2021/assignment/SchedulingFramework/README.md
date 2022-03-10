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
![](https://i.imgur.com/2SdDCGG.png)
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
            groupName nginx
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
    + Log:
    ```bash=
    $ kubectl logs {your scheduler pod name} -n kube-system
    ```
    ![](https://i.imgur.com/rq36Gje.png)

+ Input Example 2
    + Description:
    ```
        There are three jobs, red, green, yellow and the qos level is the same.
        Each job is a group consisting of 10 pods.
        Each job must have at least two pods to start scheduling.
        The order of scheduling pods is sorted by priority,
        and the priority is Red>Yellow>Green(1000>500>100).
    ```
    + Command:
    ```bash=
    $ kubectl apply -f {filename}
    ```
    + Test file [Red.yaml]:
    ```yaml=
    apiVersion: batch/v1
    kind: Job
    metadata:
    name: red
    spec:
    parallelism: 10
    completions: 10
    template:
        metadata:
        labels:
            groupName "Red"
            groupPriority: "1000"
            minAvailable: "2"
        spec:
        schedulerName: riya-scheduler
        containers:
            - name: red
            image:  riyazhu/testprogram:cpu
            resources:
                requests:
                cpu: 1101m
                memory: 262144k
                limits:
                cpu: 1101m
                memory: 262144k
        restartPolicy: Never
    ```
    + Test file [Green.yaml]:
    ```
    apiVersion: batch/v1
    kind: Job
    metadata:
    name: green
    spec:
    parallelism: 10
    completions: 10
    template:
        metadata:
        labels:
            groupName "Green"
            groupPriority: "100"
            minAvailable: "2"
        spec:
        schedulerName: riya-scheduler
        containers:
            - name: green
            image:  riyazhu/testprogram:cpu
            resources:
                requests:
                cpu: 1101m
                memory: 262144k
                limits:
                cpu: 1101m
                memory: 262144k
        restartPolicy: Never
    ```
    + Test file [Yellow.yaml]:
    ```yaml=
    apiVersion: batch/v1
    kind: Job
    metadata:
    name: yellow
    spec:
    parallelism: 10
    completions: 10
    template:
        metadata:
        labels:
            groupName "Yellow"
            groupPriority: "500"
            minAvailable: "2"
        spec:
        schedulerName: riya-scheduler
        containers:
            - name: yellow
            image:  riyazhu/testprogram:cpu
            resources:
                requests:
                cpu: 1101m
                memory: 262144k
                limits:
                cpu: 1101m
                memory: 262144k
        restartPolicy: Never
    ```
+ Output Example 2
    + Command:
    ```bash=
    $ kubectl get pod
    ```
    ![](https://i.imgur.com/6m6SLkD.png)
    + Log:
    ```bash=
    $ kubectl logs {your scheduler pod name} -n kube-system
    ```
    ![](https://i.imgur.com/yPqOvOJ.png)
+ Input Example 3
    + Description:
    ```
        This example is to check whether pods follow QOS level as the scheduling order when pods have the same priority.
        There are two jobs, hi and hello.
        The QOS level of hi is **Guaranteed**.
        The QOS level of hellow is **Best Effort**.
    ```
    + Command:
    ```bash=
    $ kubectl apply -f {filename}
    ```
    + Test file [Hi.yaml]:
    ```yaml=
    apiVersion: batch/v1
    kind: Job
    metadata:
    name: hi
    spec:
    parallelism: 10
    completions: 10
    template:
        metadata:
        labels:
            groupName "Hi"
            groupPriority: "100"
            minAvailable: "2"
        spec:
        schedulerName: riya-scheduler
        containers:
            - name: hi
            image:  riyazhu/testprogram:cpu
            # Guaranteed
            resources:
                requests:
                cpu: 1101m
                memory: 262144k
                limits:
                cpu: 1101m
                memory: 262144k
        restartPolicy: Never
    ```
    + Test file [Hello.yaml]:
    ```yaml=
    apiVersion: batch/v1
    kind: Job
    metadata:
    name: hello
    spec:
    parallelism: 10
    completions: 10
    template:
        metadata:
        labels:
            groupName "Hello"
            groupPriority: "100"
            minAvailable: "2"
        spec:
        schedulerName: riya-scheduler
        containers:
            - name: hello
            image:  riyazhu/testprogram:cpu
            # Best-Effort
            # resources:
            #   requests:
            #     cpu: 1101m
            #     memory: 262144k
            #   limits:
            #     cpu: 1101m
            #     memory: 262144k
        restartPolicy: Never
    ```
+ Output Example 3
    + Description:
    ```
        We can expect that the jobs of hi was done first.
    ```
    + Command:
    ```bash=
    $ kubectl get pod
    ```
    ![](https://i.imgur.com/yq6mibh.png)
    + Log:
    ```bash=
    $ kubectl logs {your scheduler pod name} -n kube-system
    ```
    ![](https://i.imgur.com/qFjSrEI.png)
+ Input Example 4
    + Description:
    ```
        The example you can test the pods in same group have the same priority, but have different qos level.
        The qos level of black-pusheen: **BestEffort**.
        The qos level of gray-pusheen: **Burstable**.
        The qos level of white-pusheen: **Guaranteed**.

        We can expected that the white-pusheen has the highest scheduling prioity 
        and the black-pushenn has the lowest scheduling prioity in this case.
    ```
    + Command:
    ```bash=
    $ kubectl apply -f {filename}
    ```
    + Test file:
    ```yaml=
    apiVersion: v1
    kind: Pod
    metadata:
    name: black-pusheen01
    labels:
        groupName "pusheen"
        groupPriority: "100"
        minAvailable: "2"
    spec:
    schedulerName: riya-scheduler
    containers:
    - name: black-pusheen01
        image: riyazhu/testprogram:cpu
        # BestEffort
    ---
    apiVersion: v1
    kind: Pod
    metadata:
    name: gray-pusheen01
    labels:
        groupName "pusheen"
        groupPriority: "100"
        minAvailable: "2"
    spec:
    schedulerName: riya-scheduler
    containers:
    - name: gray-pusheen01
        image: riyazhu/testprogram:cpu
        # Burstable
        resources:
        requests:
            cpu: 500m
            memory: 10000k
        limits:
            cpu: 1101m
            memory: 262144k
    ---
    apiVersion: v1
    kind: Pod
    metadata:
    name: white-pusheen01
    labels:
        groupName "pusheen"
        groupPriority: "100"
        minAvailable: "2"
    spec:
    schedulerName: riya-scheduler
    containers:
    - name: white-pusheen01
        image: riyazhu/testprogram:cpu
        # Guaranteed
        resources:
        requests:
            cpu: 1101m
            memory: 262144k
        limits:
            cpu: 1101m
            memory: 262144k
    ---
    apiVersion: v1
    kind: Pod
    metadata:
    name: black-pusheen02
    labels:
        groupName "pusheen"
        groupPriority: "100"
        minAvailable: "2"
    spec:
    schedulerName: riya-scheduler
    containers:
    - name: black-pusheen02
        image: riyazhu/testprogram:cpu
        # BestEffort
    ---
    apiVersion: v1
    kind: Pod
    metadata:
    name: white-pusheen02
    labels:
        groupName "pusheen"
        groupPriority: "100"
        minAvailable: "2"
    spec:
    schedulerName: riya-scheduler
    containers:
    - name: white-pusheen02
        image: riyazhu/testprogram:cpu
        # Guaranteed
        resources:
        requests:
            cpu: 1101m
            memory: 262144k
        limits:
            cpu: 1101m
            memory: 262144k
    ---
    apiVersion: v1
    kind: Pod
    metadata:
    name: gray-pusheen02
    labels:
        groupName "pusheen"
        groupPriority: "100"
        minAvailable: "2"
    spec:
    schedulerName: riya-scheduler
    containers:
    - name: gray-pusheen02
        image: riyazhu/testprogram:cpu
        # Burstable
        resources:
        requests:
            cpu: 500m
            memory: 10000k
        limits:
            cpu: 1101m
            memory: 262144k
    ---
    apiVersion: v1
    kind: Pod
    metadata:
    name: black-pusheen03
    labels:
        groupName "pusheen"
        groupPriority: "100"
        minAvailable: "2"
    spec:
    schedulerName: riya-scheduler
    containers:
    - name: black-pusheen03
        image: riyazhu/testprogram:cpu
        # BestEffort
    ---
    apiVersion: v1
    kind: Pod
    metadata:
    name: white-pusheen03
    labels:
        groupName "pusheen"
        groupPriority: "100"
        minAvailable: "2"
    spec:
    schedulerName: riya-scheduler
    containers:
    - name: white-pusheen03
        image: riyazhu/testprogram:cpu
        # Guaranteed
        resources:
        requests:
            cpu: 1101m
            memory: 262144k
        limits:
            cpu: 1101m
            memory: 262144k
    ---
    apiVersion: v1
    kind: Pod
    metadata:
    name: gray-pusheen03
    labels:
        groupName "pusheen"
        groupPriority: "100"
        minAvailable: "2"
    spec:
    schedulerName: riya-scheduler
    containers:
    - name: gray-pusheen03
        image: riyazhu/testprogram:cpu
        # Burstable
        resources:
        requests:
            cpu: 500m
            memory: 10000k
        limits:
            cpu: 1101m
            memory: 262144k
    ---
    ---
    apiVersion: v1
    kind: Pod
    metadata:
    name: black-pusheen04
    labels:
        groupName "pusheen"
        groupPriority: "100"
        minAvailable: "2"
    spec:
    schedulerName: riya-scheduler
    containers:
    - name: black-pusheen04
        image: riyazhu/testprogram:cpu
        # BestEffort
    ---
    apiVersion: v1
    kind: Pod
    metadata:
    name: white-pusheen04
    labels:
        groupName "pusheen"
        groupPriority: "100"
        minAvailable: "2"
    spec:
    schedulerName: riya-scheduler
    containers:
    - name: white-pusheen04
        image: riyazhu/testprogram:cpu
        # Guaranteed
        resources:
        requests:
            cpu: 1101m
            memory: 262144k
        limits:
            cpu: 1101m
            memory: 262144k
    ---
    apiVersion: v1
    kind: Pod
    metadata:
    name: gray-pusheen04
    labels:
        groupName "pusheen"
        groupPriority: "100"
        minAvailable: "2"
    spec:
    schedulerName: riya-scheduler
    containers:
    - name: gray-pusheen04
        image: riyazhu/testprogram:cpu
        # Burstable
        resources:
        requests:
            cpu: 500m
            memory: 10000k
        limits:
            cpu: 1101m
            memory: 262144k
    ---
    apiVersion: v1
    kind: Pod
    metadata:
    name: black-pusheen05
    labels:
        groupName "pusheen"
        groupPriority: "100"
        minAvailable: "2"
    spec:
    schedulerName: riya-scheduler
    containers:
    - name: black-pusheen05
        image: riyazhu/testprogram:cpu
        # BestEffort
    ---
    apiVersion: v1
    kind: Pod
    metadata:
    name: white-pusheen05
    labels:
        groupName "pusheen"
        groupPriority: "100"
        minAvailable: "2"
    spec:
    schedulerName: riya-scheduler
    containers:
    - name: white-pusheen05
        image: riyazhu/testprogram:cpu
        # Guaranteed
        resources:
        requests:
            cpu: 1101m
            memory: 262144k
        limits:
            cpu: 1101m
            memory: 262144k
    ---
    apiVersion: v1
    kind: Pod
    metadata:
    name: gray-pusheen05
    labels:
        groupName "pusheen"
        groupPriority: "100"
        minAvailable: "2"
    spec:
    schedulerName: riya-scheduler
    containers:
    - name: gray-pusheen05
        image: riyazhu/testprogram:cpu
        # Burstable
        resources:
        requests:
            cpu: 500m
            memory: 10000k
        limits:
            cpu: 1101m
            memory: 262144k
    ```
+ Output Example 4
    + Description:
    ```
        This is show you that the qos level about `Burstable` > `BestEffort`
        Note:
            Because the queue if first in first out,
            so maybe the creation of pod is too fast to observe hardly.
    ```
    + Log:
    ```bash=
    $ kubectl logs {your scheduler pod name} -n kube-system
    ```
    ![](https://i.imgur.com/fHKd6M7.png)
    ![](https://i.imgur.com/tvNFIkR.png)


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


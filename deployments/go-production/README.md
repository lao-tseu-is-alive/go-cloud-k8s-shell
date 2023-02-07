## Using Kustomize in kubectl to adapt deployment

you can adjust desired change in kustomization.yml then run :

    kubectl apply -k .

later if you want to delete everything

    kubectl delete -k .


### more information:

 + [kubectl kustomize usage](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/kustomization/)
 + [kustomize.io](https://kustomize.io/)
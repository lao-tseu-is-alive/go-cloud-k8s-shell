## using Kustomize to adapt deployment

you can adjust desired change in kustomization.yml then run :

    kubectl apply -k .

later if you want to delete everything

    kubectl delete -k .
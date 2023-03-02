kubectl apply -f manual-keycloak-install/issuer-secret.yaml
kubectl apply -f manual-keycloak-install/cluster-issuer.yaml
kubectl apply -f manual-keycloak-install/cert.yaml
kubectl apply -f manual-keycloak-install/ingress-cm.yaml
kubectl apply -f manual-keycloak-install/ingress.yaml

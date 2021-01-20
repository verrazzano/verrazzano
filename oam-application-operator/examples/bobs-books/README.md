# Bob's Books

The Bob's Books example is a set of three applications based on WebLogic, Helidon, and Coherence. The applications are specified in terms of OAM component and application configuration YAML files, and then deployed by applying those files.

## Prerequisites

You will need to fulfill the following prerequisites prior to running the example:

1. Create a Kubernetes cluster. A single node OKE cluster with VM Standard 1.4 is recommended.
1. Install the Verrazzano Application Operator.
1. Accept the license agreements for the images. In a web browser, navigate to the [Oracle Container Registry](https://container-registry.oracle.com):

    - Select **Middleware**, review, and _Sign in_ to accept the licenses for the WebLogic and Coherence images.

    - Select **Verrazzano**, review, and accept the licenses for the four repositories listed at the top of the page: `example-bobbys-coherence`, `example-bobbys-front-end`, `example-bobs-books-order-manager`, and `example-roberts-coherence`.

    You will not be able to pull these images until you have accepted the licenses.

## Deploy the Example Applications

Run the `install-bobs-books.sh` script to create all of the necessary resources and install the example applications. You need to provide credentials for both the Oracle Container Registry and GitHub Container Registry.
```
WEBLOGIC_PASS=welcome1 MYSQL_PASS=WebLogic1234 ./install-bobs-books.sh <ghcr username> <ghcr password> <ocr username> <ocr password>
```
**NOTE:** You should use more secure passwords.

Alternatively, you can specify the credentials in environment variables: `GHCR_USER`, `GHCR_PASS`, `OCR_USER` and `OCR_PASS`. When the installation completes, the script will print out URLs that you can use to access the applications.

## Detailed Steps Description

1. Create the namespace and apply the `verrazzano-managed` label.
    ```
    kubectl create namespace bobs-books
    kubectl label namespaces bobs-books verrazzano-managed=true
    ```

1. Create a secret for the Oracle Container Registry using your credentials.
    ```
    kubectl create secret docker-registry ocr --docker-server=container-registry.oracle.com --docker-username='<sso-username>' --docker-password='<sso-pw>' -n bobs-books
    ```

1. Create a secret for the GitHub Container Registry.
    ```
    kubectl create secret docker-registry github-packages --docker-username=<username> --docker-password=<password> --docker-server=ghcr.io -n bobs-books
    ```

1. Create the application secrets.
    ```
    kubectl create secret generic bobbys-front-end-weblogic-credentials --from-literal=password=welcome1 --from-literal=username=weblogic -n bobs-books

    kubectl create secret generic bobbys-front-end-runtime-encrypt-secret --from-literal=password=welcome1 -n bobs-books
    kubectl label secret bobbys-front-end-runtime-encrypt-secret weblogic.domainUID=bobbys-front-end -n bobs-books

    kubectl create secret generic bobs-bookstore-weblogic-credentials --from-literal=password=welcome1 --from-literal=username=weblogic -n bobs-books

    kubectl create secret generic bobs-bookstore-runtime-encrypt-secret --from-literal=password=welcome1 -n bobs-books
    kubectl label secret bobs-bookstore-runtime-encrypt-secret weblogic.domainUID=bobs-bookstore -n bobs-books

    kubectl create secret generic mysql-credentials \
        --from-literal=username=books \
        --from-literal=password=WebLogic1234 \
        --from-literal=url=jdbc:mysql://mysql.bobs-books.svc.cluster.local:3306/books \
        -n bobs-books
    ```
    **NOTE:** You should use more secure passwords.

1. Install Coherence.
    ```
    helm repo add coherence https://oracle.github.io/coherence-operator/charts
    helm repo update
    helm install coherence-operator coherence/coherence-operator \
       --namespace bobs-books \
       --version 2.1.1 \
       --set serviceAccount=coherence-operator
   ```

1. Install the Coherence workload and the Bob's Books OAM components and applications. This deploys the applications to the cluster.
    ```
    kubectl apply -f workload-coh.yaml
    kubectl apply -f components.yaml
    kubectl apply -f applications.yaml
    ```

1. After all of the pods are ready, expose the applications using load balancers.
    ```
    kubectl expose service robert-helidon -n bobs-books --port=8080 --target-port=8080 --type=LoadBalancer --name=robert-lb

    kubectl expose pod bobbys-front-end-managed-server1 -n bobs-books --port=8001 --target-port=8001 --type=LoadBalancer --name=bobby-lb

    kubectl expose pod bobs-bookstore-managed-server1 -n bobs-books --port=8011 --target-port=8001 --type=LoadBalancer --name=bobs-orders-lb
    ```

1. Locate the external IP address.
    ```
    kubectl get service -n "istio-system" "istio-ingressgateway" -o jsonpath={.status.loadBalancer.ingress[0].ip}

    11.22.33.44
    ```

1. Access the applications. To access the applications in a browser, you will need to do one of the following:
    * **Option 1:** Temporarily modify the `/etc/hosts` file (on Mac or Linux) or `c:\Windows\System32\Drivers\etc\hosts` file (on Windows 10), to add entries mapping the hosts used by the applications to the external IP address assigned to your gateway. For example:
      ```
      11.22.33.44 roberts-books.example.com
      11.22.33.44 bobbys-books.example.com
      11.22.33.44 bobs-orders.example.com
      ```
      a. Open a browser and navigate to the Robert's Books UI at `http://roberts-books.example.com/`.

      b. Navigate to the Bobby's Books UI at `http://bobbys-books.example.com/bobbys-front-end`.

      c. Navigate to the Bob's order manager  UI at `http://bobs-orders.example.com/bobs-bookstore-order-manager/orders`.

    * **Option 2:** Alternatively, point your own DNS name to the load balancer's external IP address. In this case, you would need to have edited the `bobs-books-app.yaml` file to use the appropriate values under the `hosts` section for each application (such as `application-host.your.domain`), before deploying the applications.
      Then, you can use a browser to access each of the applications as shown below:

      a. Robert's Books UI at `http://<your-roberts-books-host.your.domain>/`

      b. Bobby's Books UI at `http://<your-bobbys-books-host.your.domain>/bobbys-front-end`

      a. Bob's order manager UI at `http://<your-bobs-orders-host.your.domain>/`

### Robert's Books Application Details

The Robert's Books application consists of a Helidon component and a Coherence cluster component.  The Helidon component also contains the statically compiled React UI.

Creating and deploying the Robert's Books application demonstrates three types of OAM resources:

- A WorkloadDefinition YAML file needed to register the Coherence CRDs.
- OAM component definitions for the Helidon and Coherence cluster components of the Robert's Books application.
- An OAM application for the overall application deployment.

#### Workload Definition

The workload definition is simple; it just specifies the CRD that you want to use in the component, Coherence in this case:

  ```
  apiVersion: core.oam.dev/v1alpha2
  kind: WorkloadDefinition
  metadata:
    name: coherenceclusters.coherence.oracle.com
  spec:
    definitionRef:
      name: coherenceclusters.coherence.oracle.com
  ```

#### Component Definition

The component definition has the component CRD fields followed by the Workload CR fields.  You can put any Workload CR, such as Service or ContainerizedWorkload, in the component.  If you registered a CRD with a WorkloadDefinition, then you can use the YAML file for that resource.  See the `bobs-books-comp.yaml` file for all of the component definitions.

  ```
  apiVersion: core.oam.dev/v1alpha2
  kind: Component
  metadata:
    name: robert-coh
    namespace: bobs-books
  spec:
    workload:
      apiVersion: coherence.oracle.com/v1
      kind: CoherenceCluster
  ...
  ```
   
Everything below `workload` is specific to the component kind and, in this case, can be any valid Coherence YAML file.

#### Application Definition

The ApplicationConfiguration used here is the most basic; it simply lists the components that will be deployed:

  ```
  apiVersion: core.oam.dev/v1alpha2
  kind: ApplicationConfiguration
  metadata:
    name: robert
    ...
  spec:
    components:
      - componentName: robert-helidon
      - componentName: robert-coh
   ```

You can optionally use traits and scopes which will tailor the application as needed.    

#### Notes on the Coherence Component

The Coherence YAML file is slightly different from Verrazzano Bob's Books model.  The YAML file used for OAM has to specify
the Coherence cluster elements for the cluster itself.  It also specifies the application section, which has
the name of the image containing the code to load the cache with books. In Verrazzano, only the application
section is needed because the Verrazzano operator builds up the full Coherence YAML file. 

### Bobby's Books Application Details

The Bobby's Books application has four components: a WebLogic domain, a Helidon service, a Helidon pod, and a Coherence cluster.

The WebLogic component provides the UI and calls the Helidon REST API to get books. The Helidon service is needed so that the WebLogic UI can reach the Helidon component. The Helidon component gets the books out of the Coherence cache.

The WebLogic YAML for OAM is essentially identical to that in the Verrazzano model file `bobs-books-model.yaml`, except for the top section.
The difference is that the OAM component places the WebLogic CR YAML file after the `workload` element, whereas the Verrazzano model
puts it after the `domainCRValues` element.  The Helidon and Coherence components are the same as in the Robert's Books application, except the names
and images are different.  The Helidon service is a common pattern used by pods to communicate within the Kubernetes cluster.
As mentioned previously, the Coherence YAML file differs the most from the demo-model YAML file.

### Bob's Orders Application Details

The Bob's Orders application has a WebLogic domain and a MySQL database.  The WebLogic component provides the UI and manages orders in the database.

Notice that the Bob's MySQL ConfigMap YAML file (see the `bobs-mysql-configmap` component in `bobs-books-comp.yaml`) has a script that runs in the init container that creates the DB schema and writes a sample record.

# Uninstall the Example
In order to uninstall the example, run:
```
./uninstall-bobs-books.sh
```

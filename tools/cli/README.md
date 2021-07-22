# Verrazzano CLI

The Verrazzano Command Line Interface (CLI) is a tool which can be used to interact with the Verrazzano resources.

## Setting up the keycloak client
[Keycloak](https://github.com/keycloak/keycloak) provides Identity and Access Management in Verrazzano for authentication to various dashboards and the CLI application. To run the Verrazzano Console locally, first you need to configure the **verrazzano-pkce** [OpenID Connect client](https://www.keycloak.org/docs/latest/server_admin/#oidc-clients) to authenticate the login and API requests originating from the application deployed at `localhost`.

1. Access the Keycloak administration console for your Verrazzano environment: `https://keycloak.v8o-env.v8o-domain.com`
2. Log in with the Keycloak admin user and password. Typically the Keycloak admin user name is `keycloakadmin` and the password can be obtained from your management cluster:

```bash
  kubectl get secret --namespace keycloak keycloak-http -o jsonpath={.data.password} | base64 --decode; echo
```

For more information on accessing Keycloak and other user interfaces in Verrazzano, see [Get console credentials](https://github.com/verrazzano/verrazzano/blob/master/install/README.md#6-get-console-credentials).

3. Navigate to **Clients** and select the client, **verrazzano-pkce**. On the **Settings** page, go to **Valid Redirect URIs** and select the plus (+) sign to add the redirect URL `http://localhost/*`.
4. On the same page, go to **Web Origins** and select the plus (+) sign to add `http://localhost/`.
5. Click **Save**.

You can also set up a separate Keycloak client for local access using [these](https://www.keycloak.org/docs/latest/server_admin/#oidc-clients) instructions.

## Getting the Verrazzano user credentials

Verrazzano installations have a default user `verrazzano` configured in the Verrazzano Keycloak server which can be used for authentication for accessing the CLI. To get the password for the `verrazzano` user from the management cluster, run:

```bash
   kubectl get secret --namespace verrazzano-system verrazzano -o jsonpath={.data.password} | base64 --decode; echo
```

The Verrazzano Console accesses the Verrazzano API using [JSON Web Token (JWT)](https://en.wikipedia.org/wiki/JSON_Web_Token)-based authentication enabled by the [Keycloak Authorization Services](https://www.keycloak.org/docs/4.8/authorization_services/). The CLI application requests this token from the Keycloak API Server. To access the Keycloak API, the user accessing the CLI application must be logged in to Keycloak and have a valid session. When an existing Keycloak user session is expired or upon the expiration of the [refresh token](https://auth0.com/blog/refresh-tokens-what-are-they-and-when-to-use-them/), the user has to login again.

## Setting up the environment variables
````
export VZ_KEYCLOAK_URL=<your Keycloak URL> e.g. https://keycloak.default.11.22.33.44.xip.io
export VZ_CLIENT_ID=<your client id which allows redirect uri on http://localhost or verrazzano-pkce if using default>
export VZ_REALM=<your Verrazzano Realm> e.g. verrazzano-realm if you are using default
````
## Building the CLI

The binary will be located in ``/bin``.

Run ``make cli``

## Usage

Run ``vz --help`` for a list of available commands.

````
# Login into the Verrazzano API server
$ vz login  https://verrazzano.example.com
Login sucessful!

# List the clusters
$ vz cluster list
Name    Age     Status  Description
local   2d9m    Ready   Admin Cluster

# Register a managed cluster
$ vz cluster register managed1 -d "Managed Cluster" -c "ca-secret-managed1"
verrazzanomanagedcluster/managed1 created

# Fetch the manifest to be applied on the managed cluster to complete registration
$ vz cluster get-registration-manifest managed1 > managed1.yaml | kubectl --context kind-managed1 apply -f .
$ vz cluster list
Name        Age     Status  Description
local       2d9m    Ready   Admin Cluster
managed1    30s     Ready   Managed Cluster

# Create a new project
$ vz project add project1 -p "managed1"
verrazzanoproject/project1 created

# List all the projects
$ vz project list
Name        Age     Clusters    Namespaces
project1    30s     managed1    project1

# Delete the project
$ vz project delete project1
verrazzanoproject/project1 deleted

# Deregister a managed cluster
$ vz cluster deregister managed1
verrazzanomanagedcluster/managed1 deregistered

# Logout of the API server
$ vz logout
Logout successful!
````

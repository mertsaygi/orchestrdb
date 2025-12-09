# OrchestrDB Operator

OrchestrDB is a Kubernetes operator that helps you manage databases and database users in a simple and automated way.

Today, many cloud database services (like **Azure Database**) allow you to create a database *instance*, but they do not allow you to create new databases inside the instance. For example, in AWS RDS PostgreSQL, you cannot create databases through the AWS API. You must connect manually and run SQL commands.

This operator solves that problem.

With OrchestrDB:

- You can **create databases** inside an existing PostgreSQL instance.
- You can **create users** with **read-only**, **read-write**, or **owner** access.
- You can store generated user credentials in Kubernetes Secrets.
- You can manage everything declaratively using YAML.

PostgreSQL is supported today.  
Support for **MySQL**, **SQL Server**, and **Oracle** is planned.

---

## Features

### Database Management
- Create a database inside a PostgreSQL instance.
- If the database already exists, nothing breaks.
- SSL modes supported.

### User Management
- Create users with auto-generated passwords.
- Store credentials in a Secret created by the operator.
- Grant access to one or more databases.
- Roles:
  - `readonly`
  - `readwrite`
  - `owner`
- Scope:
  - `database` → grants for a single database
  - `instance` → wide permissions for the whole instance

### Secure Admin Credentials
Admin credentials can be provided in two ways:

1. Inline (not recommended for production)  
2. Via a Kubernetes Secret using `adminSecretRef`

Admin secret example:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: rds-admin
  namespace: default
type: Opaque
stringData:
  username: myadmin
  password: mypassword
```

## Install with Helm

```bash
helm repo add orchestrdb https://ghcr.io/mertsaygi/orchestrdb

helm repo update

helm install orchestrdb orchestrdb/orchestrdb \
  --namespace orchestrdb-system \
  --create-namespace
```

## Example Database Resource

```yaml
apiVersion: orchestrdb.mertsaygi.net/v1alpha1
kind: Database
metadata:
  name: appdb
  namespace: default
spec:
  host: mydb.xxxxx.eu-central-1.rds.amazonaws.com
  port: 5432
  name: appdb
  adminSecretRef:
    name: rds-admin
    userKey: username
    passwordKey: password
  sslMode: require
```

## Example User Resource

```yaml
apiVersion: orchestrdb.mertsaygi.net/v1alpha1
kind: User
metadata:
  name: appdb-user
  namespace: default
spec:
  host: mydb.xxxxx.eu-central-1.rds.amazonaws.com
  port: 5432
  username: app_user
  adminSecretRef:
    name: rds-admin
    userKey: username
    passwordKey: password
  sslMode: require
  generatedSecret:
    name: appdb-user-secret
  access:
    - dbName: appdb
      role: readwrite
      scope: database
    - dbName: auditdb
      role: readonly
      scope: database
```

## Important Notes

Secret must not exist

The Secret defined in generatedSecret must not exist before the operator runs. If it exists, the operator will fail to protect the stored credentials.

### Reconciliation

- If user or database creation fails, the operator retries.
- Updating the YAML triggers reconciliation again.

### Permissions

- User creation and grants are idempotent.
- Multiple access rules are supported.

## Limitations

- Only PostgreSQL is supported now.
- MySQL, SQL Server, and Oracle support is planned.
- Table-level permissions are not implemented yet.

## Roadmap

- MySQL adapter
- SQL Server adapter
- Oracle adapter
- Optional deletion logic on CR removal
- Metrics and dashboards

## License

MIT License. Contributions are welcome.
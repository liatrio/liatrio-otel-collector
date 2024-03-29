[comment]: <> (Code generated by mdatagen. DO NOT EDIT.)

# gitprovider

## Default Metrics

The following metrics are emitted by default. Each of them can be disabled by applying the following configuration:

```yaml
metrics:
  <metric_name>:
    enabled: false
```

### git.repository.branch.commit.aheadby.count

Number of commits a branch is ahead of the default branch

| Unit | Metric Type | Value Type |
| ---- | ----------- | ---------- |
| {branch} | Gauge | Int |

#### Attributes

| Name | Description | Values |
| ---- | ----------- | ------ |
| repository.name | The full name of the Git repository | Any Str |
| branch.name | The name of the branch in a given repository | Any Str |

### git.repository.branch.commit.behindby.count

Number of commits a branch is behind the default branch

| Unit | Metric Type | Value Type |
| ---- | ----------- | ---------- |
| {branch} | Gauge | Int |

#### Attributes

| Name | Description | Values |
| ---- | ----------- | ------ |
| repository.name | The full name of the Git repository | Any Str |
| branch.name | The name of the branch in a given repository | Any Str |

### git.repository.branch.count

Number of branches in a repository

| Unit | Metric Type | Value Type |
| ---- | ----------- | ---------- |
| {branch} | Gauge | Int |

#### Attributes

| Name | Description | Values |
| ---- | ----------- | ------ |
| repository.name | The full name of the Git repository | Any Str |

### git.repository.branch.line.addition.count

Count of lines added to code in a branch

| Unit | Metric Type | Value Type |
| ---- | ----------- | ---------- |
| {branch} | Gauge | Int |

#### Attributes

| Name | Description | Values |
| ---- | ----------- | ------ |
| repository.name | The full name of the Git repository | Any Str |
| branch.name | The name of the branch in a given repository | Any Str |

### git.repository.branch.line.deletion.count

Count of lines deleted from code in a branch

| Unit | Metric Type | Value Type |
| ---- | ----------- | ---------- |
| {branch} | Gauge | Int |

#### Attributes

| Name | Description | Values |
| ---- | ----------- | ------ |
| repository.name | The full name of the Git repository | Any Str |
| branch.name | The name of the branch in a given repository | Any Str |

### git.repository.branch.time

Time the branch has existed

| Unit | Metric Type | Value Type |
| ---- | ----------- | ---------- |
| s | Gauge | Int |

#### Attributes

| Name | Description | Values |
| ---- | ----------- | ------ |
| repository.name | The full name of the Git repository | Any Str |
| branch.name | The name of the branch in a given repository | Any Str |

### git.repository.count

Number of repositories in an organization

| Unit | Metric Type | Value Type |
| ---- | ----------- | ---------- |
| {repository} | Gauge | Int |

### git.repository.pull_request.count

The number of pull requests in a repository, categorized by their state (either open or merged)

| Unit | Metric Type | Value Type | Aggregation Temporality | Monotonic |
| ---- | ----------- | ---------- | ----------------------- | --------- |
| {pull_request} | Sum | Int | Cumulative | true |

#### Attributes

| Name | Description | Values |
| ---- | ----------- | ------ |
| state | Allows us to differentiate pull request activity within the repository | Str: ``open``, ``merged`` |
| repository.name | The full name of the Git repository | Any Str |

### git.repository.pull_request.open_time

The amount of time a pull request has been open

| Unit | Metric Type | Value Type |
| ---- | ----------- | ---------- |
| s | Gauge | Int |

#### Attributes

| Name | Description | Values |
| ---- | ----------- | ------ |
| repository.name | The full name of the Git repository | Any Str |
| branch.name | The name of the branch in a given repository | Any Str |

### git.repository.pull_request.time_to_approval

The amount of time it took a pull request to go from open to approved

| Unit | Metric Type | Value Type |
| ---- | ----------- | ---------- |
| s | Gauge | Int |

#### Attributes

| Name | Description | Values |
| ---- | ----------- | ------ |
| repository.name | The full name of the Git repository | Any Str |
| branch.name | The name of the branch in a given repository | Any Str |

### git.repository.pull_request.time_to_merge

The amount of time it took a pull request to go from open to merged

| Unit | Metric Type | Value Type |
| ---- | ----------- | ---------- |
| s | Gauge | Int |

#### Attributes

| Name | Description | Values |
| ---- | ----------- | ------ |
| repository.name | The full name of the Git repository | Any Str |
| branch.name | The name of the branch in a given repository | Any Str |

## Optional Metrics

The following metrics are not emitted by default. Each of them can be enabled by applying the following configuration:

```yaml
metrics:
  <metric_name>:
    enabled: true
```

### git.repository.contributor.count

Total number of unique contributors to a repository

| Unit | Metric Type | Value Type |
| ---- | ----------- | ---------- |
| {contributor} | Gauge | Int |

#### Attributes

| Name | Description | Values |
| ---- | ----------- | ------ |
| repository.name | The full name of the Git repository | Any Str |

## Resource Attributes

| Name | Description | Values | Enabled |
| ---- | ----------- | ------ | ------- |
| git.vendor.name | The name of the Git vendor/provider (ie. GitHub / GitLab) | Any Str | true |
| organization.name | Git Organization or Project Name | Any Str | true |

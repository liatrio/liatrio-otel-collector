# sspr

What does your receiver do?

## Features

- [x] Write a feature description here

## Getting Started

```yaml
sample: data
```

Here is a more complete example:

```yaml
receivers:
    sspr:
        sample: data
service:
    pipelines:
        metrics:
            receivers: [..., sspr]
            processors: []
            exporters: [...]
```

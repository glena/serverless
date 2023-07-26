# Simple service to deploy serverless code

Small sample service to host custom NodeJS apps. It is deployed to an EKS AWS cluster using Pulumi.


https://github.com/glena/serverless/assets/5647310/270be687-0f1f-44da-8536-12196d1a9eb2


## How to run

```
REGION= ACCESS_KEY= SECRET_KEY= go run .
```

## API

```
POST /function
{
    "Name": "...", // Script name,
    "Script": "...", // Script code,
}
```

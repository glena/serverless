# Simple service to deploy serverless code

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
# http-proxy


## Build

```bigquery
docker build -t http_proxy .
docker run -p 8080:8080 -p 8000:8000 -t http_proxy
```
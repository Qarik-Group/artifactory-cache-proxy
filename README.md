# Artifactory Cache Proxy
This project implements a HTTP proxy which caches all GET requests into Artifactory.
It can be configured to use an HTTP proxy itself to access the internet.

## Usage

Please follow [these](https://github.com/ruslo/hunter/blob/master/docs/user-guides/hunter-user/artifactory-cache-server.rst) instructions to configure your Artifactory repo.

To get a local Artifactory using docker run:

```bash
docker run --name artifactory -d \
    -p 8081:8081 docker.bintray.io/jfrog/artifactory-oss:latest
```

Now lets build and run the proxy:
```bash
./build.sh
export ARTIFACTORY_URL=http://localhost:8081/artifactory
export ARTIFACTORY_TOKEN={token from instructions above}
export ARTIFACTORY_REPO={repo name from instructions above}

# optionally configure a proxy to reach the internet
export http_proxy=http://example.com:8080
export https_proxy=http://example.com:8080
./proxy-linux &

# now update proxy config to point to the process
export http_proxy=localhost:8080
export https_proxy=localhost:8080
```

To test if it works run the following:
```bash
curl -k --proxy localhost:8080 https://example.com
# This should give you a response and trigger a caching request

curl -k --proxy localhost:8080 https://example.com
# This request should be served from artifactory
```

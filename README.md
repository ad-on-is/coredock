# 🧩 coredock

A lightweight sidecar container that automatically exposes Docker containers as DNS entries, making container discovery and inter-container communication seamless.

## ✨ Features

- Automatic DNS Registration: Exposes running Docker containers as DNS A records (e.g., containername.domain.com)
- PTR Records: Provides reverse DNS lookups for container IP addresses
- SRV Records: Exposes service discovery records for your containers
- Network Auto-Connect: Automatically connects containers to a specified Docker network
- IP Filtering: Filter exposed A records by IP prefixes to control which container IPs are published
- Custom Domains: Configure one or multiple domains for DNS resolution
- Forward queries to other hosts running coredock
- Configure containers via labels

### 💡 How It Works

coredock monitors your Docker daemon for running containers and automatically:

1. Creates DNS A records mapping container names to their IP addresses
2. Generates PTR records for reverse DNS lookups
3. Publishes SRV records for service discovery
4. Optionally connects containers to a specified network
5. Filters published IPs based on your configured prefixes

### 🌐 Use Cases

- Development Environments: Eliminate hardcoded IPs in your local Docker setup
- Service Discovery: Enable containers to find each other by name
- Microservices: Simplify inter-service communication

### ⚡ Installation

```yaml
services:
  coredock:
    image: ghcr.io/ad-on-is/coredock
    restart: always
    container_name: coredock
    environment:
      - COREDOCK_DOMAINS=docker.lan
      - COREDOCK_IP_PREFIXES=10,192 # [either] (recommended) only expose A records for these IP prefixes
      - COREDOCK_IGNORE_IP_PREFIXES=172 # [or] (recommended) ignore these IP prefixes
      - COREDOCK_NETWORKS=vlan40,vlan10 # (optional) auto-connect containers to these networks
      - COREDOCK_NAMESERVERS=10.0.0.2:53 # (optional) other coredock hosts
    volumes:
      - /var/run/docker.sock:/run/docker.sock
    ports:
      - 53:53
      - 53:53/udp
```

#### Settings

- COREDOCK_DOMAINS: comma separated list of domains to be exposed. (defaults to 'docker')
- COREDOCK_IP_PREFIXES: Containers usually have multliple IPs when assigned to multiple internal/external networks. Tell coredock to only
  use these IP prefixes. Comma separated list (i.e 10.10,192.10)
- COREDOCK_IGNORE_IP_PREFIXES: Same as above, but tell coredock to ignore these prefixes. Comma separated list (i.e. 172)
- COREDOCK_NETWORKS: Automatically assign new containers to these networks. The networks must exist prior to assigning them. Comma separated
  list (i.e. vlan40,br0.20)
- COREDOCK_NAMESERVERS: Forward queries to other nameservers. This is usesfull, if you want one main coredock service to query other
  coredock services on different hosts. Comma separated list (i.e 10.10.10.11:53)

### 🐳 Usage in containers

```yaml
services:
  app-to-ignore:
    image: example/app:latest
    container_name: app-to-ignore
    restart: always
    labels:
      coredock.ignore: true # will not be handled by coredock
  app:
    image: example/app:latest
    container_name: app
    restart: always
    labels:
      coredock.srv: 80 # will create _http._tcp.app.domain.com SRV record
      coredock.srv#api: 3000 # will create _http._tcp.api.domain.com SRV record
      coredock.srv#_http._tcp.websocket: 6000 # will create _http._tcp.websocket.domain.com SRV record
      coredock.alias: my-alias
```

#### Labels

- `coredock.ignore: true` - Ignores the container
- `coredock.srv: 3000` - Creates an SRV record with `_http._tcp.containername` pointing to port 3000
- `coredock.srv#alias: 3000` Creates an SRV record with `_http._tcp.alias` pointing to port 3000. Useful when containers come with a
  backend-service and a frontend-service running on different ports.
- `coredock.srv#_<service>._<proto>.alias: 3000` - Allows you to specify custom service and protocol for the SRV record.
- `coredock.alias: foo,bar` - Comma separated list to create CNAME records of the service.

### 🔍 DNS Queries

```bash
dig app.docker.lan
# ;; ANSWER SECTION:
# app.docker.lan.        10      IN      A       10.0.0.2

dig my-alias.docker.lan
# ;; ANSWER SECTION:
#my-alias.docker.lan.            10      IN      CNAME   app.docker.lan.
# app.docker.lan.        10      IN      A       10.0.0.2

dig _http._tcp.api.docker.lan SRV
#;; ANSWER SECTION:
# _http._tcp.api.docker.lan. 10      IN      SRV     10 5 3000 app.docker.lan.

# ;; ADDITIONAL SECTION:
# app.docker.lan.        10      IN      A       10.0.0.2

dig -x 10.0.0.2
# ;; ANSWER SECTION:
# 2.0.0.10.in-addr.arpa. 10   IN      PTR     app.docker.lan.

```

#### Example scenario: Caddyfile

Automatically proxy all requests `<name>.example.com` to `<name>.docker.lan` and their corresponding SRV port.

```Caddyfile
example.com {
  @srv_lookup {
    header_regexp host Host ^([^.]+)\.examle\.com$
  }

  handle @srv_lookup {
    reverse_proxy {
      dynamic srv _http._tcp.{http.regexp.host.1}.docker.lan
    }
  }
}

```

## License

MIT

```

```

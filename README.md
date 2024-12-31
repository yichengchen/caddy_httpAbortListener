httpAbortListener for Caddy
===========================

This package abort the connections that come on the TLS port as an HTTP request by detecting using the first few bytes that it's not a TLS handshake, but instead an HTTP request.

This plugin is inspired and modified by the [HTTPRedirectListenerWrapper](https://github.com/caddyserver/caddy/blob/master/modules/caddyhttp/httpredirectlistener.go).

## Caddy module name

```
caddy.listeners.http_abort
```


### Caddyfile Examples


```Caddyfile
{
	servers {
		listener_wrappers {
			http_abort
			tls
		}
	}
}
```

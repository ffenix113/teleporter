package web

import (
	"log"
	"net"
	"net/http"
)

func IPWhitelist(ips []string) Middleware {
	ipMap := make(map[string]struct{}, len(ips))
	for _, ip := range ips {
		ipMap[ip] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			reqAddr, _, err := net.SplitHostPort(request.RemoteAddr)
			if err != nil {
				log.Printf("could not split host:port: %s\n", err.Error())

				writer.WriteHeader(http.StatusForbidden)
				return
			}

			if _, ok := ipMap[reqAddr]; !ok {
				log.Printf("IP is not in whitelist: %q\n", reqAddr)

				writer.WriteHeader(http.StatusForbidden)
				return
			}

			next.ServeHTTP(writer, request)
		})
	}
}

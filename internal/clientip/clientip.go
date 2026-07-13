package clientip

import (
	"net/netip"

	"github.com/fox-toolkit/fox"
	"github.com/fox-toolkit/fox/clientip"
)

var DefaultResolver = newChain(
	must(clientip.NewLeftmostNonPrivate(clientip.XForwardedForKey, 15)),
	must(clientip.NewLeftmostNonPrivate(clientip.ForwardedKey, 15)),
	must(clientip.NewSingleIPHeader(fox.HeaderXRealIP)),
	must(clientip.NewSingleIPHeader(fox.HeaderCFConnectionIP)),
	must(clientip.NewSingleIPHeader(fox.HeaderTrueClientIP)),
	must(clientip.NewSingleIPHeader(fox.HeaderFastClientIP)),
	must(clientip.NewSingleIPHeader(fox.HeaderXAzureClientIP)),
	must(clientip.NewSingleIPHeader(fox.HeaderXAzureSocketIP)),
	must(clientip.NewSingleIPHeader(fox.HeaderXAppengineRemoteAddr)),
	must(clientip.NewSingleIPHeader(fox.HeaderFlyClientIP)),
	clientip.NewRemoteAddr(),
)

type chain struct {
	resolvers []fox.ClientIPResolver
}

func newChain(resolvers ...fox.ClientIPResolver) chain {
	return chain{resolvers: resolvers}
}

// ClientIP try to derive the client IP using this resolver chain.
func (s chain) ClientIP(c *fox.Context) (netip.Addr, error) {
	var lastErr error
	for _, sub := range s.resolvers {
		ipAddr, err := sub.ClientIP(c)
		if err == nil {
			return ipAddr, nil
		}
		lastErr = err
	}

	return netip.Addr{}, lastErr
}

func must(resolver fox.ClientIPResolver, err error) fox.ClientIPResolver {
	if err != nil {
		panic(err)
	}
	return resolver
}

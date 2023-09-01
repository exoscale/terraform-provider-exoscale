package dns

import "github.com/exoscale/egoscale/v3/oapi"

type DNSIface interface {
}

// DNS provides access to [Exoscale DNS] API resources.
//
// [Exoscale DNS]: https://community.exoscale.com/documentation/dns/
type DNS struct {
	oapiClient *oapi.ClientWithResponses
}

// NewDNS initializes DNS with provided oapi Client.
func NewDNS(c *oapi.ClientWithResponses) DNSIface {
	return &DNS{c}
}

//func (a *DNS) Domains() *Domains {
//return NewDomains(a.oapiClient)
//}

func NewMockDNS() DNSIface {
	return &DNS{}
}

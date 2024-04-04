package endpoint

import (
	"encoding/base64"
	"fmt"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"os"
)

const testCert = `-----BEGIN CERTIFICATE-----
MIIChjCCAeigAwIBAgIUHYRJHPNW+eqW3TkSaWhpRxqyk68wCgYIKoZIzj0EAwIw
VDELMAkGA1UEBhMCUlUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoMGElu
dGVybmV0IFdpZGdpdHMgUHR5IEx0ZDENMAsGA1UEAwwEVGVzdDAgFw0yMzA4MTcx
NDE0MTRaGA8yMDUxMDEwMjE0MTQxNFowVDELMAkGA1UEBhMCUlUxEzARBgNVBAgM
ClNvbWUtU3RhdGUxITAfBgNVBAoMGEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDEN
MAsGA1UEAwwEVGVzdDCBmzAQBgcqhkjOPQIBBgUrgQQAIwOBhgAEADrZB/oUNXuU
kAoyC1DCoqWnp0pdJx5GuxqxAJD9uMYOS05G3PjAboesJohnoFGOld2Zh2Kuj6OJ
ULh9hTj14eB7AZT4YX/vjA/odBS/Bu9PSjMiyrwTCms1hkMl2EvS06Hc3ElrjsuY
YMma/Chd8G+GAX12ijNO7BMlhLjhoZm383oao1MwUTAdBgNVHQ4EFgQU3x7cM6Kd
TEJN6KQvc0cHjAODOCwwHwYDVR0jBBgwFoAU3x7cM6KdTEJN6KQvc0cHjAODOCww
DwYDVR0TAQH/BAUwAwEB/zAKBggqhkjOPQQDAgOBiwAwgYcCQUtlwuBJgT4gSGfH
yax9nYcFz6DzTaXWe3CZG0oLReUTrP88CeYfevWAvO7etL8IRKr48OWWm+sARDzY
GH/IDRigAkIBI45wN1CUGzzBjF8/faxNy6XWhcSkFZW7oCRR0MWaL6bn69naej8K
0msNdKBh0Uyk4SK0q+4NlBMTgoimpXcNdk8=
-----END CERTIFICATE-----`

type MockClient struct {
	RealClient
}

func (c MockClient) PeerAdd(wgPub wgtypes.Key, params map[string]string) (APIResponse, error) {
	params["peer_add"] = base64.StdEncoding.EncodeToString(wgPub[:])
	c.url.RawQuery = c.addQueryParams(params)

	fmt.Fprintf(os.Stderr, "Test Request: %s\n", c.url.String())

	return APIResponse{
		Code:                     "0",
		OpenvpnClientCertificate: testCert,
	}, nil
}

package storage

func GetEndpointHost(brigade *Brigade, user *User) string {
	if user.EndpointDomain != "" {
		return user.EndpointDomain
	}
	if brigade.EndpointDomain != "" {
		return brigade.EndpointDomain
	}
	return brigade.EndpointIPv4.String()
}

package http

func SessionsSetRequestsToEndpointsSet(ss SessionSet) map[string]Endpoints {
	epMap := map[string]Endpoints{}
	for sessionId, ses := range ss.SessionMap {
		epMap[sessionId] = SessionToEndpoints(ses)
	}
	return epMap
}

func SessionToEndpoints(ses Session) Endpoints {
	eps := NewEndpoints()
	for _, req := range ses.RequestSet.Requests {
		pathMethod := req.URLPattern + " " + req.Method
		ep, ok := eps.EndpointsMap[pathMethod]
		if !ok {
			ep = Endpoint{
				Method: req.Method,
				URL:    req.URLPattern}
		}
		ep.AddStatus(req.StatusCode, req.SubStatusCode, req.Time)
		eps.EndpointsMap[pathMethod] = ep
	}
	return eps
}

//Endpoint
//StatusTimes []

/*
func WriteSessionsSetToCSV() {
SessionId
RequestId
	PathMethod
Time
Status


}

*/

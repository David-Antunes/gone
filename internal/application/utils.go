package application

func sniffSocketPath(id string) string {
	return "/tmp/" + id + ".sniff"
}
func interceptSocketPath(id string) string {
	return "/tmp/" + id + ".intercept"
}

func getInterceptId(id string, direction bool) string {
	if direction {
		return id + "-tx"
	} else {
		return id + "-rx"
	}
}

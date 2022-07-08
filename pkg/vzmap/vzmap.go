package vzmap

//UnionStringMaps returns the union of m1 and m2. Key collisions favor m2.
func UnionStringMaps(m1, m2 map[string]string) map[string]string {
	u := map[string]string{}
	for k, v := range m1 {
		u[k] = v
	}
	for k, v := range m2 {
		u[k] = v
	}
	return u
}

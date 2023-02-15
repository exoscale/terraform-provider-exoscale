package general

import "fmt"

type ResourceIDStringer interface {
	Id() string
}

func ResourceIDString(d ResourceIDStringer, name string) string {
	id := d.Id()
	if id == "" {
		id = "<new resource>"
	}

	return fmt.Sprintf("%s (ID = %s)", name, id)
}

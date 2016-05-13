package core

/*import (
	"core/config"
)*/

type APP struct {
	App_id       string
	App_name     string
	App_domain   string
	App_cat      []int32
	App_ver      string
	App_bundle   string
	App_paid     string
	App_kw       string
	App_storeurl string
}

func NewAPP() *APP {

	return &APP{}
}

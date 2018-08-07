package error

import "fmt"

type Collector []error

func (c *Collector) Collect(e error) { *c = append(*c, e) }

func (c *Collector) Error() (err string) {
	err = "Collected errors:\n"
	for i, e := range *c {
		err += fmt.Sprintf("\tError %d: %s\n", i, e.Error())
	}

	return err
}

func (c Collector) HasError() bool {
	return len(c) > 0
}

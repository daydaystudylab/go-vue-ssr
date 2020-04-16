// Code generated by go-vue-ssr: https://github.com/zbysir/go-vue-ssr

package main

func NewRender() *Render {
	r := newRender()
	r.Components = map[string]ComponentFunc{
		"component": r.Component_component,
		"info":      r.Component_info,
		"page":      r.Component_page,
		"slot":      r.Component_slot,
		"v-on":      r.Component_vOn,
		"vOn":       r.Component_vOn,
	}
	return r
}

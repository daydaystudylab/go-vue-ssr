// Code generated by go-vue-ssr: https://github.com/bysir-zl/go-vue-ssr

package tplgo

func NewRender() *Render {
	r := &Render{}
	r.Components = map[string]ComponentFunc{
		"bench":      r.Component_bench,
		"class":      r.Component_class,
		"component":  r.Component_component,
		"directive":  r.Component_directive,
		"helloworld": r.Component_helloworld,
		"page":       r.Component_page,
		"slot":       r.Component_slot,
		"text":       r.Component_text,
		"v-for":      r.Component_vFor,
		"vFor":       r.Component_vFor,
		"vif":        r.Component_vif,
		"vtext":      r.Component_vtext,
		"x-slot":     r.Component_xSlot,
		"x-style":    r.Component_xStyle,
		"xSlot":      r.Component_xSlot,
		"xStyle":     r.Component_xStyle,
		"xattr":      r.Component_xattr,
	}
	return r
}

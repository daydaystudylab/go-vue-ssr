// Code generated by go-vue-ssr: https://github.com/bysir-zl/go-vue-ssr
// src_hash:8b44a03ecdd41a904eeb3d85fb46c047

package tplgo

func (r *Render) Component_vFor(options *Options) string {
	this := extendMap(r.Prototype, options.Props)
	_ = this
	return r.Tag("div", true, &Options{
		Slot: map[string]NamedSlotFunc{"default": func(props map[string]interface{}) string {
			return func() string {
				var c = ""

				for index, item := range interface2Slice(lookInterface(this, "list")) {
					c += func(xdata map[string]interface{}) string {
						this := extendMap(xdata, map[string]interface{}{
							"index": index,
							"item":  item,
						})

						return "<span>" + interfaceToStr(lookInterface(this, "index"), true) + ": " + interfaceToStr(lookInterface(this, "item"), true) + " " + r.Component_slot(&Options{
							Slot: map[string]NamedSlotFunc{"default": func(props map[string]interface{}) string { return "" }},
							P:    options,
							Data: this,
						}) + "</span>"
					}(this)
				}
				return c
			}()
		}},
		P:    options,
		Data: this,
	})
}

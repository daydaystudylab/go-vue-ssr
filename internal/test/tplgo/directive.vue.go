// Code generated by go-vue-ssr: https://github.com/bysir-zl/go-vue-ssr
// src_hash:95c5c3a698b94d1f9f8cca22fe95d5ed

package tplgo

func (r *Render) Component_directive(options *Options) string {
	this := extendMap(r.Prototype, options.Props)
	_ = this
	return r.Tag("div", true, &Options{
		Slot: map[string]NamedSlotFunc{"default": func(props map[string]interface{}) string {
			return func() string {
				if interfaceToBool(lookInterface(this, "show")) {
					return r.Tag("p", false, &Options{
						Slot: map[string]NamedSlotFunc{"default": func(props map[string]interface{}) string { return "\n        test animate\n      " }},
						P:    options,
						Directives: []directive{
							{Name: "v-animate", Value: map[string]interface{}{"type": "up"}, Arg: ""},
						},
						Data: this,
					})
				} else {
					return "\n      !show\n    "
				}
				return ""
			}() + r.Tag("p", false, &Options{
				Slot: map[string]NamedSlotFunc{"default": func(props map[string]interface{}) string { return "" }},
				P:    options,
				Directives: []directive{
					{Name: "v-set", Value: map[string]interface{}{"id": lookInterface(this, "id"), "value": map[string]interface{}{"swiper": map[string]interface{}{"speed": lookInterface(this, "speed")}}}, Arg: "swiper"},
				},
				Data: this,
			}) + r.Tag("p", false, &Options{
				Slot: map[string]NamedSlotFunc{"default": func(props map[string]interface{}) string { return "" }},
				P:    options,
				Directives: []directive{
					{Name: "v-get", Value: nil, Arg: ""},
				},
				Data: this,
			}) + "\n    333\n  "
		}},
		P: options,
		Directives: []directive{
			{Name: "v-animate", Value: map[string]interface{}{"time": "5s", "xclass": lookInterface(this, "xclass")}, Arg: ""},
		},
		Data: this,
	})
}
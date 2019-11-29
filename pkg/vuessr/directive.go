package vuessr

import (
	"encoding/xml"
	"fmt"
	"github.com/bysir-zl/vue-ssr/pkg/vuessr/ast_from_api"
	"strings"
)

// 指令:
// 指令会影响当前节点的渲染, 返回修改后的go代码
// 有一个特殊的指令: v-slot, 会将节点代码改为空, 并且写入到namedSlotCode里.
type Directive interface {
	Exec(e *VueElement, code string) (resCode string, namedSlotCode map[string]string)
}

type Directives map[string]Directive

func (d Directives) Exec(e *VueElement, code string) (descCode string, namedSlotCode map[string]string) {
	namedSlotCode = map[string]string{}
	for _, v := range d {
		var n2 map[string]string
		code, n2 = v.Exec(e, code)
		for k, v := range n2 {
			namedSlotCode[k] = v
		}
	}
	return code, namedSlotCode
}

type VForDirective struct {
	arrayKey string
	itemKey  string
	indexKey string
}

func (e VForDirective) Exec(el *VueElement, code string) (descCode string, namedSlotCode map[string]string) {
	vfArray := e.arrayKey
	vfItem := e.itemKey
	vfIndex := e.indexKey
	// 将自己for, 将子代码的data字段覆盖, 实现作用域的修改
	return fmt.Sprintf(`
func ()string{
  var c = ""

  for index, item := range lookInterfaceToSlice(%s, "%s") {
    c += func(xdata map[string]interface{}) string{
        %s := extendMap(map[string]interface{}{
          "%s": index,
          "%s": item,
        }, xdata)

        return %s
    }(%s)
  }
return c
}()`, DataKey, vfArray, DataKey, vfIndex, vfItem, code, DataKey), nil
}

type VIfDirective struct {
	condition string
}

func (e VIfDirective) Exec(el *VueElement, code string) (descCode string, namedSlotCode map[string]string) {
	condition, err := ast_from_api.JsCode2Go(e.condition, DataKey)
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf(`
func ()string{
  if interfaceToBool(%s) {return %s}
  return ""
}()`, condition, code), nil
}

type VSlotDirective struct {
	slotName string
	propsKey string
}

func (e VSlotDirective) Exec(el *VueElement, code string) (descCode string, namedSlotCode map[string]string) {
	// 插槽支持传递props, 需要有自己的作用域, 所以需要使用闭包实现
	code = fmt.Sprintf(`
func(props map[string]interface{}) string{
	%s := extendMap(map[string]interface{}{"%s": props}, %s)
	return %s
}`, DataKey, e.propsKey, DataKey, code)

	namedSlotCode = map[string]string{
		e.slotName: code,
	}

	// 插槽会将原来的子代码去掉, 并将代码放在namedSlot里.
	descCode = `""`
	return
}

// raw: 指令的值
func getVForDirective(attr xml.Attr) (d VForDirective) {
	val := attr.Value

	ss := strings.Split(val, " in ")
	d.arrayKey = strings.Trim(ss[1], " ")

	left := strings.Trim(ss[0], " ")
	// (item, index) in list
	if strings.Contains(left, ",") {
		left = strings.Trim(left, "()")
		ss := strings.Split(left, ",")
		d.itemKey = strings.Trim(ss[0], " ")
		d.indexKey = strings.Trim(ss[1], " ")

	} else {
		d.itemKey = left
		d.indexKey = "$index"
	}

	return
}

// raw: 指令的值
func getVIfDirective(attr xml.Attr) (d VIfDirective) {
	d.condition = strings.Trim(attr.Value, " ")
	return
}

// slot可以传递props, 为了解决这个问题, 可以使用func (slotCode map[string]string, name string, propsKey string, props map[string]interface{}){}()闭包实现
func getVSlotDirective(attr xml.Attr) (d VSlotDirective) {
	d.slotName = attr.Name.Local
	d.propsKey = attr.Value
	// 不应该为空, 否则可能会导致生成的go代码有误
	if d.propsKey == "" {
		d.propsKey = "slotProps"
	}

	return
}
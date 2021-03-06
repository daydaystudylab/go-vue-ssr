package vuessr

import (
	"github.com/zbysir/go-vue-ssr/pkg/vuessr/parser"
	"strings"
)

type VueElement struct {
	// 是否是root节点
	// 正常情况下template下的第一个节点是root节点, 如 template > div.
	// 如果没有按照vue组件的写法来写组件(template下只能有一个元素), 则所有元素都不会被当为root节点
	//
	// 是否是根节点, 指的是<template>下一级节点, 这个节点会继承父级传递下来的class/style
	IsRoot           bool
	NodeType         parser.NodeType
	TagName          string
	Text             string
	DocType          string
	Attrs            []Attribute       // 除去指令/props/style/class之外的属性
	Directives       []Directive       // 自定义指令, 运行时
	ElseIfConditions []ElseIf          // 将与if指令匹配的elseif/else关联在一起
	Class            []string          // 静态class
	Style            map[string]string // 静态style
	StyleKeys        []string          // 样式的key, 用来保证顺序
	Props            Props             // props, 包括动态的class和style
	Children         []*VueElement     // 子节点
	VIf              *VIf              // 处理v-if需要的数据
	VFor             *VFor
	VSlot            *VSlot
	VElse            bool // 如果是VElse节点则不会生成代码(而是在vif里生成代码)
	VElseIf          bool
	// v-html / v-text
	// 支持v-html / v-text指令覆盖子级内容的组件有: template / html基本标签
	// component/slot和自定义组件不支持(没有必要)v-html/v-text覆盖子级
	VHtml string
	VText string
	VOn   []VOnDirective // v-on与普通自定义指令不同，其中表达式不会去调用方法，而是存储调用的方法和args然后生成js代码
}

type Attribute struct {
	Key, Val string
}

type Directive struct {
	Name  string // v-animate
	Value string // {'a': 1}
	Arg   string // v-set:arg
}

// v-on:click="buttonClick(args1, args2)" // 方法（参数） 支持：在这种类型上，所有的参数都是读取props值。
// v-on:click="function(){a=a+1}" // js方法 不支持：表达式中没办法准确的识别变量是模板传递的还是js中的。
//  如a+1中我们无法得知a到底是读取props(翻译成go代码)还是使用全局的js变量（不翻译）。
// v-on:click="a=a+1" // 表达式 不支持：同上
type VOnDirective struct {
	Func  string // buttonClick
	Args  string // args1, args2, 将被翻译成go。
	Exp   string // 原始表达式: buttonClick(args1, args2)
	Event string // click
}

type ElseIf struct {
	Types      string // else / elseif
	Condition  string // elseif语句的condition表达式
	VueElement *VueElement
}

type VIf struct {
	Condition string // 条件表达式
	ElseIf    []*ElseIf
}

func (p *VIf) AddElseIf(v *ElseIf) {
	p.ElseIf = append(p.ElseIf, v)
}

type VFor struct {
	ArrayKey string
	ItemKey  string
	IndexKey string
}

type VSlot struct {
	SlotName string
	PropsKey string
}

func (p Props) Omit(key ...string) Props {
	kMap := map[string]struct{}{}
	for _, k := range key {
		kMap[k] = struct{}{}
	}

	a := Props{}
	for _, item := range p {
		if _, ok := kMap[item.Key]; ok {
			continue
		}
		a = append(a, item)
	}
	return a
}

func ParseVue(filename string) (v *VueElement, err error) {
	htmlParser := parser.GoHtml{}

	es, err := htmlParser.Parse(filename)
	if err != nil {
		return
	}

	p := VueElementParser{}
	if len(es) == 1 {
		v = p.Parse(es[0])

		// 和vue不同的是, 在根template下的所有子节点都是root节点
		// 这样可以实现在组件上方添加一些指令, 而不破坏组件
		if v.TagName == "template" {
			for _, v := range v.Children {
				v.IsRoot = true
			}
		}
	} else {
		// 如果是多个节点, 则自动添加template包裹, 作为入口
		// 这种情况下不会存在root节点
		e := &parser.Element{
			TagName:  "template",
			NodeType: parser.ElementNode,
			Children: es,
		}
		v = p.Parse(e)
	}
	return
}

type VueElementParser struct {
}

func (p VueElementParser) Parse(e *parser.Element) *VueElement {
	vs := p.parseList([]*parser.Element{e})
	return vs[0]
}

// 递归处理同级节点
// 使用数组有一个好处就是方便的处理串联的v-if
func (p VueElementParser) parseList(es []*parser.Element) []*VueElement {
	vs := make([]*VueElement, len(es))

	var ifVueEle *VueElement
	for i, e := range es {
		var props []Prop
		var ds []Directive
		var vOn []VOnDirective
		var class []string
		style := map[string]string{}
		var styleKeys []string
		var attrs []Attribute
		var vIf *VIf
		var vFor *VFor
		var vSlot *VSlot

		// 标记节点是不是if
		var vElse *ElseIf
		var vElseIf *ElseIf

		var vHtml string
		var vText string

		for _, attr := range e.Attrs {
			oriKey := attr.Key
			ss := strings.Split(oriKey, ":")
			nameSpace := "-"
			key := oriKey
			if len(ss) == 2 {
				key = ss[1]
				nameSpace = ss[0]
			}

			if nameSpace == "v-bind" || nameSpace == "" {
				// v-bind & shorthands :
				props = append(props, Prop{
					Key: key,
					Val: attr.Val,
				})
			} else if strings.HasPrefix(oriKey, "@") || nameSpace == "v-on" {
				// v-on & shorthands @
				// v-on和普通的指令不同, 它的值是一个方法, 并且是js方法, 所以在模板中无法计算或者存储该值, 只能换一个方法: 存储为对象{event, funcName}, 让js代码再去调用.
				end := strings.LastIndex(attr.Val, ")")
				start := strings.Index(attr.Val, "(")
				// func(a, b)
				if end != -1 && start != -1 {
					args := attr.Val[start+1 : end]
					fun := attr.Val[:start]

					event := strings.TrimPrefix(key, "@")

					vOn = append(vOn, VOnDirective{
						Func:  fun,
						Args:  args,
						Event: event,
						Exp:   attr.Val,
					})
				} else {
					// func
					event := strings.TrimPrefix(key, "@")
					vOn = append(vOn, VOnDirective{
						Func:  attr.Val,
						Args:  "",
						Event: event,
						Exp:   attr.Val,
					})
				}
			} else if strings.HasPrefix(oriKey, "v-") {
				// 指令
				// v-if=""
				// v-slot:name=""
				// v-else-if=""
				// v-else
				// v-html
				switch {
				case key == "v-for":
					val := attr.Val

					ss := strings.Split(val, " in ")
					arrayKey := strings.Trim(ss[1], " ")

					left := strings.Trim(ss[0], " ")
					var itemKey string
					var indexKey string
					// (item, index) in list
					if strings.Contains(left, ",") {
						left = strings.Trim(left, "()")
						ss := strings.Split(left, ",")
						itemKey = strings.Trim(ss[0], " ")
						indexKey = strings.Trim(ss[1], " ")
					} else {
						// (item) or item
						left = strings.Trim(left, "()")
						itemKey = left
						indexKey = "$index"
					}

					vFor = &VFor{
						ArrayKey: arrayKey,
						ItemKey:  itemKey,
						IndexKey: indexKey,
					}
				case key == "v-if":
					vIf = &VIf{
						Condition: strings.Trim(attr.Val, " "),
						ElseIf:    nil,
					}
				case nameSpace == "v-slot":
					slotName := key
					propsKey := attr.Val
					// 不应该为空, 否则可能会导致生成的go代码有误
					if propsKey == "" {
						propsKey = "slotProps"
					}
					vSlot = &VSlot{
						SlotName: slotName,
						PropsKey: propsKey,
					}
				case key == "v-else-if":
					vElseIf = &ElseIf{
						Types:     "elseif",
						Condition: strings.Trim(attr.Val, " "),
					}
				case key == "v-else":
					vElse = &ElseIf{
						Types:     "else",
						Condition: strings.Trim(attr.Val, " "),
					}
				case key == "v-html":
					vHtml = strings.Trim(attr.Val, " ")
				case key == "v-text":
					vText = strings.Trim(attr.Val, " ")
				default:
					// 自定义指令
					var name string
					var arg string
					if nameSpace != "-" {
						name = nameSpace
						arg = key
					} else {
						name = key
					}
					ds = append(ds, Directive{
						Name:  name,
						Value: strings.Trim(attr.Val, " "),
						Arg:   arg,
					})
				}
			} else if attr.Key == "class" {
				ss := strings.Split(attr.Val, " ")
				for _, v := range ss {
					if v != "" {
						class = append(class, v)
					}
				}
			} else if attr.Key == "style" {
				ss := strings.Split(attr.Val, ";")
				for _, v := range ss {
					v = strings.Trim(v, " ")
					ss := strings.Split(v, ":")
					if len(ss) != 2 {
						continue
					}
					key := strings.Trim(ss[0], " ")
					val := strings.Trim(ss[1], " ")
					style[key] = val
					styleKeys = append(styleKeys, key)
				}
			} else {
				key := attr.Key
				if attr.Namespace != "" {
					key = attr.Namespace + ":" + attr.Key
				}
				attrs = append(attrs, Attribute{
					Key: key,
					Val: attr.Val,
				})
			}
		}

		ch := p.parseList(e.Children)

		v := &VueElement{
			IsRoot:           false,
			NodeType:         e.NodeType,
			TagName:          e.TagName,
			Text:             e.Text,
			DocType:          e.DocType,
			Attrs:            attrs,
			Directives:       ds,
			ElseIfConditions: []ElseIf{},
			Class:            class,
			Style:            style,
			StyleKeys:        styleKeys,
			Props:            props,
			Children:         ch,
			VIf:              vIf,
			VFor:             vFor,
			VSlot:            vSlot,
			VElse:            vElse != nil,
			VElseIf:          vElseIf != nil,
			VHtml:            vHtml,
			VText:            vText,
			VOn:              vOn,
		}

		// 记录vif, 接下来的elseif将与这个节点关联
		if vIf != nil {
			ifVueEle = v
		} else {
			// 如果有vif环境了, 但是中间跳过了, 则需要取消掉vif环境 (v-else 必须与v-if 相邻)
			skipNode := e.NodeType == parser.CommentNode
			if !skipNode && vElse == nil && vElseIf == nil {
				ifVueEle = nil
			}
		}

		if vElseIf != nil {
			if ifVueEle == nil {
				panic("v-else-if must below v-if")
			}
			vElseIf.VueElement = v
			ifVueEle.VIf.AddElseIf(vElseIf)
		}
		if vElse != nil {
			if ifVueEle == nil {
				panic("v-else must below v-if")
			}
			vElse.VueElement = v
			ifVueEle.VIf.AddElseIf(vElse)
			ifVueEle = nil
		}

		vs[i] = v
	}

	return vs
}

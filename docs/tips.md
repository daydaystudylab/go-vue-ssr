# Tips

这里会写一些值得注意的提示, 包括:
- 与Vue特性有差异的地方
- 有趣的例子

## Component
所有参与编译的vue文件都会被注册为组件, 组件名字就是文件名, 故不要取重复的文件名.

文件名的kebab-case写法与PascalCase写法是一样的, 同时 <my-component-name> 和 <MyComponentName>都能正常使用.

和vue组件不同的是, Go-vue-ssr为了简化逻辑, html页面也被当成了组件, 如下模板也是能够正常被渲染的.
```vue
<!DOCTYPE html>
<html lang="zh">
<head>
  <meta charset="UTF-8">
  <title>{{title}}</title>
</head>
<body>
  <h1 v-html="title" style="text-align: center; margin-top: 100px"></h1>
</body>
</html>
```

不过对于不满足Vue组件规范的组件就不会有Class/Style的组件特性: [Class and Style Bindings#With-Components](https://vuejs.org/v2/guide/class-and-style.html#With-Components)

## Props
由于不支持像Vue一样声明props, 所以所有v-bind写法都会被传递到组件内部. 

所有作用在基础html标签的props都会被渲染为attr.

作用在自定义组件的props默认不会被渲染为attr, 如果需要一部分props被渲染成attrs, 可以在render.CanBeAttr(TODO ^_^)中修改这个行为.

## CustomDirectives
功能和VueSSR中的[指令](https://ssr.vuejs.org/guide/universal.html#custom-directives)类似

在自定义指令中, 你可以操作组件的渲染行为, 如添加一个Class/Style, 或者修改子节点.

当然没有虚拟节点之后能够操作的数据是有限的.

下面是使用指令实现的一个功能: 渲染多个Swiper组件.
 
原理是利用指令将多个组件的数据收集起来, 供给Js处理.

编写两个自定义指令:
```go
package demo

type MyRender struct{
	Render vuetpl.Render
	Ctx map[string][]string
}

func (r *MyRender) addDirective() {
	render:= r.Render
    // v-set指令会在渲染过程中存储变量, 可以通过v-set获取变量
	// 使用闭包特性在多个指令中共享数据
	// 语法: v-set:swiper="{a: 1}"
    render.Directive("v-set", func(binding vuetpl.DirectivesBinding, options *vuetpl.Options) {
        r.Ctx[binding.Arg] = append(r.Ctx[binding.Arg], binding.Value)
    })
    // 语法: v-get={'swiper': 'global'}
    // 对象中key表示获取哪一个key, value表示将存储为的变量名
    render.Directive("v-get", func(binding vuetpl.DirectivesBinding, options *vuetpl.Options) {
        options.Slot["default"] = func(props map[string]interface{}) string {
            m := binding.Value.(map[string]interface{})
            var sortKey []string
            for k := range m {
                sortKey = append(sortKey, k)
            }

            sort.Strings(sortKey)
            str := ""

            for _, k := range sortKey {
                v := m[k]

                bs, _ := json.Marshal(r.Ctx[k])
                str += fmt.Sprintf("var %s = %s;", v, string(bs))
            }

            return str
        }
    })
}
```

编写swiper.vue组件如下:
```vue
<!-- swiper.vue -->
<template>
    <div class="swiper" :id="id" v-set:swiper="{id: id, speed: swiperOptions.speed, loop: true}">
        <div v-for="item in list">
            xxxx
        </div>
    </div>
</template>
```
v-set指令会在组件渲染的时候执行, 并将speed和loop数据保存下来, 现在你可以在其他地方 如页面的底部来获取刚才保存的数据
```vue
<div class="body">
</div>
<div class="footer">
    <script v-get="{'swiper': 'swiper'}"></script>
    <script>console.log(swiper)</script>
</div>
```
控制台中会打印出下面的数据, 这个数据就可以给Swiper插件使用.
```
[{id: 1, speed: "5s", loop: true}]
```
如何定义与处理数据完全取决与你.

---

另外 和vue不同, Govuessr中指令可以作用在template上, 但由于template没有真实dom, 所以无法操作如class/style等dom相关的数据, 但可以操作如data/slot这样的渲染有关的数据.

所以现在你可以实现声明变量了.

指令如下
```
render.Directive("v-data-set", func(b vuetpl.DirectivesBinding, options *vuetpl.Options) {
		options.Data[b.Arg] = b.Value
	})
```

模板如下
```
...
<template v-data-set:msg="'hello'"/>
{{msg}}
```
在第三行就可以打印出msg的值.

## Prototype
我们知道在Vue中有Store给我们提供了访问全局数据的解决方案, 那么在这个框架中如何读取全局变量呢?

你可以向render.Prototype(姑且叫这个名字)添加变量(包括方法), 其中的变量就能在所有组件中被访问到.

同时 在插值\{\{}}中使用到的函数也可以在这里定义.

```go
r := vuetpl.Render()
r.Prototype = map[string]interface{}{
    "version": "1",
    "getTag": func (args ...interface{}) interface{}{
    	return args[0].(string)+".0.0"
    } 
}
```
现在在任何组件中都可以使用到这些全局变量
```vue
<template>
    <div>
        Version: \{\{version}} <br/>
        Tag: \{\{getTag(version)}}
    </div>
</template>
```

## v-on
这个指令是运行时指令，大体功能和上面说的v-set自定义指令类似，都是存储数据，唯一不同的是v-on指令会自动生成一个event-id在dom上，用于事件与dom的绑定。

例子请看 [example/helloworld/vue/v-on.vue](/example/helloworld/vue/v-on.vue)

tip: 因为Govuessr更关注dom, 而非js, 所以这只是一个简易实现, 后续可能会优化.

------

**下一篇: [编译](genera.md)**

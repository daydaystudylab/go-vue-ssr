package vuessr

import (
	"bytes"
	"context"
	"fmt"
	"github.com/radovskyb/watcher"
	"github.com/zbysir/go-vue-ssr/internal/pkg/errors"
	"github.com/zbysir/go-vue-ssr/internal/pkg/log"
	"github.com/zbysir/go-vue-ssr/internal/version"
	"go/format"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func genComponentRenderFunc(c *Compiler, pkgName, name string, file string, srcHash string) []byte {
	ve, err := ParseVue(file)
	code := `""`
	if err != nil {
		log.Warningf("parseVue err: %v, file: %v", err, file)
	} else {
		code, _ = c.GenEleCode(ve)
		code = minifyCode(code)
	}

	f := []byte(fmt.Sprintf("// Code generated by go-vue-ssr: https://github.com/zbysir/go-vue-ssr\n// src_hash:%s\n\n"+
		"package %s\n\n"+
		"import (\"strings\")\ntype _ strings.Builder\n"+
		"func xx_%s(r *Render, w Writer, options *Options){\n"+
		"%s:= extendScope(r.Global, options.Props.data)\n"+
		"options.Directives.Exec(r, options)\n"+
		"_ = %s\n"+
		"%s\n"+
		"return"+
		"}", srcHash, pkgName, name, ScopeKey, ScopeKey, code))
	f2, err := format.Source(f)
	if err != nil {
		log.Errorf("format.Source [%s] err:%+v, src:%s", name, err, f)
		return f
	}

	return f2
}

func minifyCode(code string) string {
	// 如果前后两个都是字符串, 则可以将中间的w.WriterString删除
	// before:
	// 	 w.WriteString("<title>")
	//	 w.WriteString("" + interfaceToStr(scope.Get("title"), true) + "")
	// after:
	//   w.WriteString("<title>" + interfaceToStr(scope.Get("title"), true) + "")

	code = strings.Replace(code, "\")\nw.WriteString(\"", "", -1)
	code = strings.Replace(code, "\n\"\"\n", "\n", -1)

	// 处理多余的纯字符串拼接: "a"+"b" => "ab"
	code = strings.Replace(code, `"+"`, "", -1)

	return code
}

func tuoFeng2SheXing(src string) (outStr string) {
	l := len(src)
	var out []byte
	for i := 0; i < l; i = i + 1 {
		// 大写变小写
		if 97-32 <= src[i] && src[i] <= 122-32 {
			if i != 0 {
				out = append(out, '-')
			}
			out = append(out, src[i]+32)
		} else {
			out = append(out, src[i])
		}
	}

	return string(out)
}

func sheXing2TuoFeng(src string) (outStr string) {
	l := len(src)
	out := make([]byte, l)

	// 首字母
	out[0] = src[0]

	del := 0
	for i := 1; i < l; i = i + 1 {
		// 是下划线
		if '-' == src[i] {
			// 下划线的下一个是小写字母
			if 97 <= src[i+1] && src[i+1] <= 122 {
				out[i-del] = src[i+1] - 32
			} else {
				out[i-del] = src[i+1]
			}
			del++
			i++
		} else {
			out[i-del] = src[i]
		}
	}
	out = out[0 : l-del]
	return string(out)
}

func genCreator(components map[string]string, pkgName string) []byte {
	m := map[string]string{}
	for tagName, comName := range components {
		m[tagName] = fmt.Sprintf(`xx_%s`, comName)
	}

	f := []byte(fmt.Sprintf("// Code generated by go-vue-ssr: https://github.com/zbysir/go-vue-ssr\n\n"+
		"package %s\n\n"+
		"func NewRenderCreator() *RenderCreator{"+
		"r:=newRenderCreator()\n"+
		"r.Components = %s\n"+
		"return r"+
		"}",
		pkgName, mapGoCodeToCode(m, "ComponentFunc", true)))

	formatted, err := format.Source(f)
	if err != nil {
		log.Errorf("format.Source [%s] err:%+v, src:%s", "genCreator", err, f)
		return f
	}

	return formatted
}

// 组件名字, 驼峰
func componentName(src string) string {
	return sheXing2TuoFeng(src)
}

type VueFile struct {
	ComponentName string // xText
	Path          string
	Filename      string // x-text.vue
}

// 生成并写入文件夹
func GenAllFile(src, desc string, pkg string) (err error) {
	// 生成文件夹
	err = os.MkdirAll(desc, os.ModePerm)
	if err != nil {
		return
	}

	// 老的.vue.go文件
	oldComp, err := walkDir(desc, ".vue.go")
	if err != nil {
		return
	}

	oldVs := map[string]VueFile{}
	for _, v := range oldComp {
		_, fileName := filepath.Split(v)
		name := componentName(strings.TrimSuffix(fileName, ".vue.go"))
		oldVs[name] = VueFile{
			ComponentName: name,
			Path:          v,
			Filename:      fileName,
		}
	}

	// 生成新的组件文件
	vueFiles, err := walkDir(src, ".vue")
	if err != nil {
		return
	}

	c := NewCompiler()

	var vs []VueFile
	for _, v := range vueFiles {
		_, fileName := filepath.Split(v)
		name := componentName(strings.TrimSuffix(fileName, ".vue"))

		vs = append(vs, VueFile{
			ComponentName: name,
			Path:          v,
			Filename:      fileName,
		})

		// 注册vue组件代码
		c.AddComponent(name)
	}

	_, pkgName := filepath.Split(desc)
	if pkg != "" {
		pkgName = pkg
	}

	// 生成new代码
	code := genCreator(c.Components, pkgName)
	err = ioutil.WriteFile(desc+string(os.PathSeparator)+"creator.go", code, 0666)
	if err != nil {
		return
	}

	willDelOld := oldVs

	// 生成vue组件代码
	for _, v := range vs {
		vuePath := v.Path
		// 读取文件是否改变
		// 只有改变过才会再次编译，优化性能
		srcHash := fileMd5(vuePath, version.Version)

		codePath := desc + string(os.PathSeparator) + v.ComponentName + ".vue.go"

		if _, ok := oldVs[v.ComponentName]; ok {
			oldCode, _ := ioutil.ReadFile(codePath)
			oldCodeStr := string(oldCode)
			if strings.Contains(oldCodeStr, "src_hash:") {
				oldSrcHash := strings.Split(strings.Split(oldCodeStr, "src_hash:")[1], "\n")[0]

				if oldSrcHash == srcHash {
					// 如果hash相同，则不动老代码
					delete(willDelOld, v.ComponentName)
					continue
				}
			}
		}

		newCode := genComponentRenderFunc(c, pkgName, v.ComponentName, v.Path, srcHash)

		if _, ok := oldVs[v.ComponentName]; ok {
			// 如果有新代码则不删除老代码, 要么覆盖, 要么不动(新老代码一样)
			delete(willDelOld, v.ComponentName)
			oldCode, err := ioutil.ReadFile(codePath)
			if err != nil {
				if !os.IsNotExist(err) {
					return errors.NewCoder(err, "read oldCode file")
				}
			}

			// 对比老代码, 如果新老代码一样则不动作, 否则删除掉老代码, 新写代码
			if bytes.Equal(oldCode, newCode) {
				continue
			}

			err = ioutil.WriteFile(codePath, newCode, os.ModePerm)
			if err != nil {
				return errors.NewCoder(err, "write oldCode file")
			}
		} else {
			err = ioutil.WriteFile(codePath, newCode, os.ModePerm)
			if err != nil {
				return errors.NewCoder(err, "write oldCode file")
			}
		}
	}

	// 删除应该删除的老文件
	for _, v := range willDelOld {
		err = os.Remove(v.Path)
		if err != nil {
			err = errors.NewCoder(err, fmt.Sprintf("del oldCode file :%s", v.Path))
			return
		}
	}

	// builtin代码
	code = []byte(fmt.Sprintf("// Code generated by go-vue-ssr: https://github.com/zbysir/go-vue-ssr\n\npackage %s\n", pkgName) +
		strings.ReplaceAll(builtinCode, "package xxx", ""))
	err = ioutil.WriteFile(desc+string(os.PathSeparator)+"builtin.go", code, 0666)
	if err != nil {
		return
	}

	return
}

func GenAllFileWithWatch(ctx context.Context, src, desc string, pkg string) (err error) {
	log.Infof("watching dir and subdirectories: %s", src)

	w := watcher.New()

	err = w.AddRecursive(src)
	if err != nil {
		return
	}

	w.SetMaxEvents(1)
	// Only files that match the regular expression during file listings
	// will be watched.
	r := regexp.MustCompile(".vue$")
	w.AddFilterHook(watcher.RegexFilterHook(r, false))

	go w.Start(400 * time.Millisecond)
	defer w.Close()

	for {
		select {
		case <-ctx.Done():
			return
		case err = <-w.Error:
			return
		case e, ok := <-w.Event:
			if ok {
				log.Infof("file changed: %v", e.Path)
				err = GenAllFile(src, desc, pkg)
				if err != nil {
					return
				}
				log.Infof("compile success")
			} else {
				return
			}
		}
	}
}

func walkDir(dirPth string, suffix string) (files []string, err error) {
	files = make([]string, 0, 30)

	err = filepath.Walk(dirPth, func(filename string, fi os.FileInfo, err error) error {
		//遍历目录
		if err != nil {
			return err
		}
		if fi.IsDir() {
			// 忽略目录
			return nil
		}

		if strings.HasSuffix(filename, suffix) {
			files = append(files, filename)
		}

		return nil
	})

	return
}

func fileMd5(filePath string, salt string) string {
	oldCode, err := ioutil.ReadFile(filePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return ""
		}
		panic(err)
	}
	return Md5String(string(oldCode) + salt)
}

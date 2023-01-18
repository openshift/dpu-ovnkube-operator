// Copyright 2019 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package translate

import (
	"reflect"
	"strings"

	"github.com/coreos/ignition/v2/config/util"
	"github.com/coreos/vcontext/path"
	"github.com/coreos/vcontext/report"
)

// fieldName returns the name uses when (un)marshalling a field. t should be a reflect.Value of a struct,
// index is the field index, and tag is the struct tag used when (un)marshalling (e.g. "json" or "yaml")
func fieldName(t reflect.Value, index int, tag string) string {
	f := t.Type().Field(index)
	if tag == "" {
		return f.Name
	}
	return strings.Split(f.Tag.Get(tag), ",")[0]
}

func prefixPath(p path.ContextPath, prefix ...interface{}) path.ContextPath {
	return path.New(p.Tag, prefix...).Append(p.Path...)
}

func prefixPaths(ps []path.ContextPath, prefix ...interface{}) []path.ContextPath {
	ret := []path.ContextPath{}
	for _, p := range ps {
		ret = append(ret, prefixPath(p, prefix...))
	}
	return ret
}

func getAllPaths(v reflect.Value, tag string) []path.ContextPath {
	k := v.Kind()
	t := v.Type()
	switch {
	case util.IsPrimitive(k):
		return nil
	case k == reflect.Ptr:
		if v.IsNil() {
			return nil
		}
		return getAllPaths(v.Elem(), tag)
	case k == reflect.Slice:
		ret := []path.ContextPath{}
		for i := 0; i < v.Len(); i++ {
			paths := getAllPaths(v.Index(i), tag)
			if len(paths) > 0 {
				// struct, pointer to struct, etc.; add children
				ret = append(ret, prefixPaths(paths, i)...)
			} else {
				// primitive type; add slice entry
				ret = append(ret, path.New(tag, i))
			}
		}
		return ret
	case k == reflect.Struct:
		ret := []path.ContextPath{}
		for i := 0; i < t.NumField(); i++ {
			name := fieldName(v, i, tag)
			field := v.Field(i)
			if t.Field(i).Anonymous {
				ret = append(ret, getAllPaths(field, tag)...)
			} else {
				ret = append(ret, prefixPaths(getAllPaths(field, tag), name)...)
				ret = append(ret, path.New(tag, name))
			}
		}
		return ret
	default:
		panic("Encountered types that are not the same when they should be. This is a bug, please file a report")
	}
}

// Return a copy of the report, with the context paths prefixed by prefix.
func prefixReport(r report.Report, prefix interface{}) report.Report {
	var ret report.Report
	ret.Merge(r)
	for i := range ret.Entries {
		entry := &ret.Entries[i]
		entry.Context = path.New(entry.Context.Tag, prefix).Append(entry.Context.Path...)
	}
	return ret
}

// Utility function to run a translation and prefix the resulting
// TranslationSet and Report.
func Prefixed(tr Translator, prefix interface{}, from interface{}, to interface{}) (TranslationSet, report.Report) {
	tm, r := tr.Translate(from, to)
	return tm.Prefix(prefix), prefixReport(r, prefix)
}

// Utility function to run a translation and merge the result, with the
// specified prefix, into the specified TranslationSet and Report.
func MergeP(tr Translator, tm TranslationSet, r *report.Report, prefix interface{}, from interface{}, to interface{}) {
	translations, report := tr.Translate(from, to)
	tm.MergeP(prefix, translations)
	r.Merge(prefixReport(report, prefix))
}

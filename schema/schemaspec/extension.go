package schemaspec

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

// Remainer is the interface that is implemented by types that can store
// additional attributes and children resources.
type Remainer interface {
	// Remain returns a resource representing any extra children and attributes
	// that are related to the struct but were not mapped to any of its fields.
	Remain() *Resource
}

// DefaultExtension can be embedded in structs that need basic default behavior.
// For instance, DefaultExtension implements Remainer, and has a private *Resource
// field that can store additional attributes and children that do not match the
// structs fields.
type DefaultExtension struct {
	Extra Resource
}

// Remain implements the Remainer interface.
func (d *DefaultExtension) Remain() *Resource {
	return &d.Extra
}

type registry map[string]interface{}

var (
	extensions   = make(registry)
	extensionsMu sync.RWMutex
)

func (r registry) lookup(ext interface{}) (string, bool) {
	extensionsMu.RLock()
	defer extensionsMu.RUnlock()
	for k, v := range r {
		if reflect.TypeOf(ext) == reflect.TypeOf(v) {
			return k, true
		}
	}
	return "", false
}

// Register records the type of ext in the global extension registry.
// If Register is called twice with the same name or if ext is nil,
// it panics.
func Register(name string, ext interface{}) {
	extensionsMu.Lock()
	defer extensionsMu.Unlock()
	if ext == nil {
		panic("schemaspec: Register extension is nil")
	}
	if _, dup := extensions[name]; dup {
		panic("schemaspec: Register called twice for type " + name)
	}
	extensions[name] = ext
}

// As reads the attributes and children resources of the resource into the target struct.
func (r *Resource) As(target interface{}) error {
	if err := validateStructPtr(target); err != nil {
		return err
	}
	existingAttrs, existingChildren := existingElements(r)
	var seenName bool
	v := reflect.ValueOf(target).Elem()
	for _, ft := range specFields(target) {
		field := v.FieldByName(ft.field)
		switch {
		case ft.isName:
			if seenName {
				return errors.New("schemaspec: extension must have only one isName field")
			}
			seenName = true
			if field.Kind() != reflect.String {
				return errors.New("schemaspec: extension isName field must be of type string")
			}
			field.SetString(r.Name)
		case hasAttr(r, ft.tag):
			attr, _ := r.Attr(ft.tag)
			if err := setField(field, attr); err != nil {
				return err
			}
			delete(existingAttrs, attr.K)
		case isResourceSlice(field.Type()):
			if err := setChildSlice(field, childrenOfType(r, ft.tag)); err != nil {
				return err
			}
			delete(existingChildren, ft.tag)
		case isSingleResource(field.Type()):
			c := childrenOfType(r, ft.tag)
			if len(c) == 0 {
				continue
			}
			res := c[0]
			n := reflect.New(field.Type().Elem())
			ext := n.Interface()
			if err := res.As(ext); err != nil {
				return err
			}
			field.Set(n)
			delete(existingChildren, ft.tag)
		}
	}
	rem, ok := target.(Remainer)
	if !ok {
		return nil
	}
	extras := rem.Remain()
	for attrName := range existingAttrs {
		attr, ok := r.Attr(attrName)
		if !ok {
			return fmt.Errorf("schemaspec: expected attr %q to exist", attrName)
		}
		extras.SetAttr(attr)
	}
	for childType := range existingChildren {
		children := childrenOfType(r, childType)
		extras.Children = append(extras.Children, children...)
	}
	return nil
}

func validateStructPtr(target interface{}) error {
	typeOf := reflect.TypeOf(target)
	if typeOf.Kind() != reflect.Ptr {
		return errors.New("schemaspec: expected target to be a pointer")
	}
	if typeOf.Elem().Kind() != reflect.Struct {
		return errors.New("schemaspec: expected target to be a pointer to a struct")
	}
	return nil
}

func existingElements(r *Resource) (attrs, children map[string]struct{}) {
	attrs, children = make(map[string]struct{}), make(map[string]struct{})
	for _, ea := range r.Attrs {
		attrs[ea.K] = struct{}{}
	}
	for _, ec := range r.Children {
		children[ec.Type] = struct{}{}
	}
	return
}

func setChildSlice(field reflect.Value, children []*Resource) error {
	if field.Type().Kind() != reflect.Slice {
		return fmt.Errorf("schemaspec: expected field to be of kind slice")
	}
	typ := field.Type().Elem()
	slc := reflect.MakeSlice(reflect.SliceOf(typ), 0, len(children))
	for _, c := range children {
		n := reflect.New(typ.Elem())
		ext := n.Interface()
		if err := c.As(ext); err != nil {
			return err
		}
		slc = reflect.Append(slc, reflect.ValueOf(ext))
	}
	field.Set(slc)
	return nil
}

func setField(field reflect.Value, attr *Attr) error {
	switch field.Kind() {
	case reflect.String:
		s, err := attr.String()
		if err != nil {
			return fmt.Errorf("schemaspec: value of attr %q cannot be read as string: %w", attr.K, err)
		}
		field.SetString(s)
	case reflect.Int:
		i, err := attr.Int()
		if err != nil {
			return fmt.Errorf("schemaspec: value of attr %q cannot be read as integer: %w", attr.K, err)
		}
		field.SetInt(int64(i))
	case reflect.Bool:
		b, err := attr.Bool()
		if err != nil {
			return fmt.Errorf("schemaspec: value of attr %q cannot be read as bool: %w", attr.K, err)
		}
		field.SetBool(b)
	case reflect.Ptr:
		field.Set(reflect.ValueOf(attr.V))
	default:
		return fmt.Errorf("schemaspec: unsupported field kind %q", field.Kind())
	}
	return nil
}

// Scan reads the Extension into the Resource. Scan will override the Resource
// name or type if they are set for the extension.
func (r *Resource) Scan(ext interface{}) error {
	if lookup, ok := extensions.lookup(ext); ok {
		r.Type = lookup
	}
	v := reflect.ValueOf(ext).Elem()
	for _, ft := range specFields(ext) {
		field := v.FieldByName(ft.field)
		switch {
		case ft.isName:
			if field.Kind() != reflect.String {
				return errors.New("schemaspec: extension name field must be string")
			}
			r.Name = field.String()
		case isResourceSlice(field.Type()):
			for i := 0; i < field.Len(); i++ {
				ext := field.Index(i).Interface()
				child := &Resource{}
				if err := child.Scan(ext); err != nil {
					return err
				}
				child.Type = ft.tag
				r.Children = append(r.Children, child)
			}
		case isSingleResource(field.Type()):
			if field.IsNil() {
				continue
			}
			ext := field.Interface()
			child := &Resource{}
			if err := child.Scan(ext); err != nil {
				return err
			}
			child.Type = ft.tag
			r.Children = append(r.Children, child)
		case field.Kind() == reflect.Ptr:
			if field.IsNil() {
				continue
			}
			if field.Elem().Type() != reflect.TypeOf(LiteralValue{}) {
				return fmt.Errorf("schemaspec: pointer to non LiteralValue")
			}
			v := field.Elem().FieldByName("V").String()
			r.SetAttr(&Attr{
				K: ft.tag,
				V: &LiteralValue{V: v},
			})
		default:
			if err := scanAttr(ft.tag, r, field); err != nil {
				return err
			}
		}
	}
	rem, ok := ext.(Remainer)
	if !ok {
		return nil
	}
	extra := rem.Remain()
	for _, attr := range extra.Attrs {
		r.SetAttr(attr)
	}
	r.Children = append(r.Children, extra.Children...)
	return nil
}

func scanAttr(key string, r *Resource, field reflect.Value) error {
	var lit string
	switch field.Kind() {
	case reflect.String:
		lit = strconv.Quote(field.String())
	case reflect.Int:
		lit = fmt.Sprintf("%d", field.Int())
	case reflect.Bool:
		lit = strconv.FormatBool(field.Bool())
	default:
		return fmt.Errorf("schemaspec: unsupported field kind %q", field.Kind())
	}
	r.SetAttr(&Attr{
		K: key,
		V: &LiteralValue{V: lit},
	})
	return nil
}

// specFields uses reflection to find struct fields that are tagged with "spec"
// and returns a list of mappings from the tag to the field name.
func specFields(ext interface{}) []fieldTag {
	t := reflect.TypeOf(ext)
	var fields []fieldTag
	for i := 0; i < t.Elem().NumField(); i++ {
		f := t.Elem().Field(i)
		lookup, ok := f.Tag.Lookup("spec")
		if !ok {
			continue
		}
		parts := strings.Split(lookup, ",")
		fields = append(fields, fieldTag{
			field:  f.Name,
			tag:    lookup,
			isName: len(parts) > 1 && parts[1] == "name",
		})
	}
	return fields
}

type fieldTag struct {
	field, tag string
	isName     bool
}

func childrenOfType(r *Resource, typ string) []*Resource {
	var out []*Resource
	for _, c := range r.Children {
		if c.Type == typ {
			out = append(out, c)
		}
	}
	return out
}

func isSingleResource(t reflect.Type) bool {
	if t.Kind() != reflect.Ptr {
		return false
	}
	elem := t.Elem()
	if elem.Kind() != reflect.Struct {
		return false
	}
	for i := 0; i < elem.NumField(); i++ {
		f := elem.Field(i)
		_, ok := f.Tag.Lookup("spec")
		if ok {
			return true
		}
	}
	return false
}

func isResourceSlice(t reflect.Type) bool {
	if t.Kind() != reflect.Slice {
		return false
	}
	return isSingleResource(t.Elem())
}

func hasAttr(r *Resource, name string) bool {
	_, ok := r.Attr(name)
	return ok
}

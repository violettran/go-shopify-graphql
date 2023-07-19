package jsonutil_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/gempages/go-shopify-graphql/jsonutil"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestJSONUtil(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "JSON Util Suite")
}

var _ = Describe("UnmarshalGraphQL", func() {
	Context("builtin types", func() {
		It("unmarshal correctly", func() {
			m := map[string]any{
				"int":    -1,
				"string": "s",
				"uint":   1,
				"bool":   true,
				"float":  1.0,
				"time":   "2023-07-19T18:02:00Z",
			}
			s := struct {
				Int    int       `json:"int"`
				String string    `json:"string"`
				Uint   uint      `json:"uint"`
				Bool   bool      `json:"bool"`
				Float  float64   `json:"float"`
				Time   time.Time `json:"time"`
			}{}
			err := jsonutil.ConvertMapToStruct(m, reflect.ValueOf(&s))
			Expect(err).NotTo(HaveOccurred())
			Expect(s.Int).To(Equal(-1))
			Expect(s.String).To(Equal("s"))
			Expect(s.Uint).To(Equal(uint(1)))
			Expect(s.Bool).To(BeTrue())
			Expect(s.Float).To(Equal(float64(1)))
			Expect(s.Time).NotTo(BeNil())
		})
	})
})

type Interf interface{}

type Int int

type InterImpl struct {
	F1 string `json:"f1"`
	F2 string `json:"f2"`
}

type ConcreteStruct struct {
	Boolean    bool    `json:"boolean"`
	Number     uint    `json:"number"`
	NullString *string `json:"null_string"`
}

type Root struct {
	Int             Int               `json:"int,omitempty"`
	Struct          ConcreteStruct    `json:"struct,omitempty"`
	StructPtr       *ConcreteStruct   `json:"struct_ptr,omitempty"`
	NullString      *string           `json:"null_string,omitempty"`
	NullInt         *Int              `json:"null_int,omitempty"`
	StringSlice     []string          `json:"string_slice,omitempty"`
	NullStringSlice []*string         `json:"null_string_slice,omitempty"`
	StructSlice     []ConcreteStruct  `json:"struct_slice,omitempty"`
	NullStructSlice []*ConcreteStruct `json:"null_struct_slice,omitempty"`
	I               Interf            `json:"i,omitempty"`
}

func TestConvertMapToStruct(t *testing.T) {
	m := map[string]any{
		"int": 1,
		"struct": map[string]any{
			"boolean":     true,
			"number":      2,
			"null_string": "not null",
		},
		"struct_ptr": map[string]any{
			"boolean": true,
			"number":  3,
		},
		"null_string":       "null_string",
		"null_int":          4,
		"string_slice":      []string{"s1", "s2"},
		"null_string_slice": []*string{nil, aws.String("sss")},
		"struct_slice": []ConcreteStruct{{
			Boolean: true,
			Number:  5,
		}, {
			Boolean: true,
			Number:  6,
		}},
		"null_struct_slice": []*ConcreteStruct{{
			Boolean: true,
			Number:  7,
		}, {
			Boolean: true,
			Number:  8,
		}},
		"i": map[string]any{
			"f1": "1",
			"f2": "2",
		},
	}

	root := &Root{}
	err := jsonutil.ConvertMapToStruct(m, reflect.ValueOf(root))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if root.Int == 0 {
		t.Error("Int is 0")
		t.Fail()
	}
	if !root.Struct.Boolean {
		t.Error("Struct.Boolean is false")
		t.Fail()
	}
	if root.Struct.Number == 0 {
		t.Error("Struct.Number is 0")
		t.Fail()
	}
	if root.Struct.NullString == nil {
		t.Error("Struct.NullString is nil")
		t.Fail()
	}

	if root.StructPtr == nil {
		t.Error("StructPtr is nil")
		t.Fail()
	} else {
		if !root.StructPtr.Boolean {
			t.Error("StructPtr.Boolean is false")
			t.Fail()
		}
		if root.StructPtr.Number == 0 {
			t.Error("StructPtr.Number is 0")
			t.Fail()
		}
	}

	if root.NullString == nil {
		t.Error("NullString is nil")
		t.Fail()
	}
	if root.NullInt == nil {
		t.Error("NullInt is nil")
		t.Fail()
	}

	if len(root.StringSlice) == 0 {
		t.Error("StringSlice is empty")
		t.Fail()
	} else if root.StringSlice[0] != "s1" {
		t.Error("StringSlice[0] is not s1")
		t.Fail()
	}

	if len(root.NullStringSlice) == 0 {
		t.Error("NullStringSlice is empty")
		t.Fail()
	} else {
		if root.NullStringSlice[0] != nil {
			t.Error("NullStringSlice[0] is not nil")
			t.Fail()
		}
		if root.NullStringSlice[1] == nil {
			t.Error("NullStringSlice[1] is nil")
			t.Fail()
		} else if *root.NullStringSlice[1] != "sss" {
			t.Error("NullStringSlice[1] is not sss")
			t.Fail()
		}
	}

	if root.I == nil {
		t.Error("I is nil")
		t.Fail()
	}
}

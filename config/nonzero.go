package config

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// CheckNonZeroFormat enumerates and recursively checks o for zero values,
// populating them with defaults when configured.
//
// When zero values without defaults are observed, each of them are
// coalesced into an error message using dot notation.
func CheckNonZeroFormat(o interface{}) (err error) {
	if names, ok := NonZero(o); !ok {
		err = errors.New(
			fmt.Sprintf(
				"%s is missing required values:\n\n%s",
				reflect.TypeOf(o).Elem().Name(),
				"- "+strings.Join(names, "\n- ")))
	}
	return err
}

// NonZero accepts structs and iterates each underlying
// field to determine if its value is zero.
//
// Only fields tagged with `nonzero:""` are be evaluated.
//
// # Return Values
//
// - names is a slice of field names found to be zero.
// - ok indicates if no zero value fields were found.
//
// # Name Formatting
//
// Each value in the names slice is dot separated for each
// recursive step between struct values, e.g., Form.Name.Last
// could indicate that an input form is missing the last
// name value.
func NonZero(o any) (names []string, ok bool) {

	//===================
	// PREPARE FOR CHECKS
	//===================

	v := reflect.ValueOf(o)

	if v.Kind().String() != "ptr" {
		panic("NonZero requires pointer values")
	}

	// Work from the underlying element
	//
	// This ensures we can set values via reflect.
	v = v.Elem()
	t := v.Type()

	// Work only on structs
	if t.Kind().String() != "struct" {
		panic("NonZero works only on structs")
	}

	//===============
	// ENSURE NONZERO
	//===============

	names, ok = checkNonZero(&t, &v)

	// Construct dotted references to zero values
	if !ok {
		for i := 0; i < len(names); i++ {
			names[i] = fmt.Sprintf("%s.%s", t.Name(), names[i])
		}
	}

	return names, ok
}

// checkNonZero is a recursive function called by NonZero. It
// uses reflect methods to perform checks on each field.
func checkNonZero(t *reflect.Type, v *reflect.Value) (zNames []string, ok bool) {

	// Names that are zero value.
	zNames = []string{}

	for ind := 0; ind < v.NumField(); ind++ {

		var tag string
		var iOk bool

		tF := (*t).Field(ind)
		if tag, iOk = tF.Tag.Lookup("nonzero"); !iOk {
			continue
		}

		vF := v.Field(ind)

		//=============================
		// VALIDATE/SET THE FIELD VALUE
		//=============================

		if vF.Kind().String() == "struct" {

			//==============================
			// RECURSIVELY MANAGE THE STRUCT
			//==============================

			// Next type to check
			nT := vF.Type()

			// Recursively check all fields
			if iZ, ok := checkNonZero(&nT, &vF); !ok {
				for i := 0; i < len(iZ); i++ {
					iZ[i] = fmt.Sprintf("%s.%s", tF.Name, iZ[i])
				}
				zNames = append(zNames, iZ...)
			}

		} else if vF.IsZero() && tag != "" {

			//==================
			// SET DEFAULT VALUE
			//==================

			switch vF.Type().Kind().String() {
			case "string":

				vF.SetString(tag)
				continue

			case "uint16":

				if v, err := strconv.ParseUint(tag, 10, 16); err != nil {
					panic(err)
				} else {
					vF.SetUint(v)
				}
				continue

			case "uint8":

				if v, err := strconv.ParseUint(tag, 10, 8); err != nil {
					panic(err)
				} else {
					vF.SetUint(v)
				}
				continue

			case "bool":

				if v, err := strconv.ParseBool(tag); err != nil {
					panic(err)
				} else {
					vF.SetBool(v)
				}
				continue

			}

		} else if vF.IsZero() {

			//================================================
			// CAPTURE NAMES OF FIELDS THAT SHOULD HAVE VALUES
			//================================================

			zNames = append(zNames, tF.Name)
			continue

		}

	}

	ok = len(zNames) == 0
	return zNames, ok
}

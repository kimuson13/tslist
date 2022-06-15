package tslist

import "fmt"

func typeSetPrint(vs VisitorResult) string {
	if len(vs.TypeSets) == 0 {
		return "empty"
	}
	tslist := make([]string, 0, len(vs.TypeSets))
	for _, v := range vs.TypeSets {
		tslist = append(tslist, v.Name)
	}
	return fmt.Sprintf("%v", tslist)
}

func methodListPrint(vs VisitorResult) []string {
	methodList := make([]string, 0, len(vs.Methods))
	for _, method := range vs.Methods {
		format := fmt.Sprintf("%s", method.Name)
		format = addValues(format, method.Args)

		if len(method.Outputs) == 1 {
			if method.Outputs[0].Name == "" {
				format += addValue(format, method.Outputs[0])
			} else {
				format += " ("
				format += addValue(format, method.Outputs[0])
				format += ")"
			}
		}
		if len(method.Outputs) > 1 {
			format = addValues(format, method.Outputs)
		}
		methodList = append(methodList, format)
	}

	return methodList
}

func addValues(format string, tvs []TypeValue) string {
	format += "("
	for i, tv := range tvs {
		if i == len(tvs)-1 {
			format += addValue(format, tv)
		} else {
			format += fmt.Sprintf("%s, ", addValue(format, tv))
		}
	}
	format += ") "
	return format
}

func addValue(format string, typeValue TypeValue) string {
	if typeValue.Name == "" {
		return typeValue.Type.String()
	}

	return fmt.Sprintf("%s %s", typeValue.Name, typeValue.Type.String())
}

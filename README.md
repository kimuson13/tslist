# tslist
tslist shows the list about types satisfied by type sets
## Caution
## Install
## Situation to use
### type sets satisfy in 
## How to use
### tslist
### tslist.InterfaceVisitor
You input *ast.InterfaceType to tslist.InterfaceVisitor, so you got type sets and method list.
The return value type is `VisitorResult`. This struct has 4 fields below.
| field name | type | about |
| ---- | ---- | ---- | 
|  Pos  |  token.Pos(int)  | interface declaration position |
|  Name  |  string  | interface name(e.g. Stringer) |
| TypeSets | []TypeValue | type sets |
| Methods | []Method | method list |

TypeValue:
| field name | type | about |
| ---- | ---- | ---- |
| Name | string | identifier name. not type name |
| Type | types.Type | type's info |

**in golang.org/x/exp/typeparams implemented same APIs.**  
So, after Go future updates, typeparms will be in stadard Go statistic check package such as types.When that actually happens this API(tslist.InterfaceVisitor) will become duplicated
## Demo(tslist)
package initializers

import (
	"auth/models"
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
)

var E *casbin.Enforcer

func InitCasbin() {
	m := model.NewModel()
	m.AddDef("r", "r", "sub, obj, act")
	m.AddDef("p", "p", "sub, obj, act, eft")
	m.AddDef("g", "g", "_, _")
	m.AddDef("e", "e", "some(where (p.eft == allow)) && !some(where (p.eft == deny))")
	m.AddDef("m", "m", "g(r.sub, p.sub) && pathMatch(r.obj, p.obj) && methodMatch(r.act, p.act)")

	a, _ := gormadapter.NewAdapterByDBWithCustomTable(DB, &models.CasbinRule{}, "casbin_rule")

	E, _ = casbin.NewEnforcer(m, a)
	E.AddFunction("pathMatch", PathMatchFunc)
	E.AddFunction("methodMatch", MethodMatchFunc)
	E.LoadPolicy()
	E.AddPolicy("admin", "/api/bg/**", "ANY", "allow")
	E.AddPolicy("admin", "/api/ui/**", "ANY", "allow")
	E.AddPolicy("user", "/api/ui/**", "ANY", "allow")
}

func PathMatch(key1 string, key2 string) bool {
	// 1. key2 like "/a/*", key1 like "/a/b"
	// 2. key2 like "/a/**", key1 like "/a/b/c"

	i := strings.LastIndex(key2, "/")
	if i == -1 {
		return false
	}
	switch key2[i+1:] {
	case "*":
		return strings.HasPrefix(key1, key2[:i+1]) && !strings.Contains(key1[i+1:], "/")
	case "**":
		return strings.HasPrefix(key1, key2[:i+1])
	default:
		return key1 == key2
	}
}

func PathMatchFunc(args ...interface{}) (interface{}, error) {
	name1 := args[0].(string)
	name2 := args[1].(string)

	return PathMatch(name1, name2), nil
}

func MethodMatch(key1 string, key2 string) bool {
	// Method type: GET, POST, PUT, DELETE, ANY
	if key2 == "ANY" {
		return true
	} else {
		return key1 == key2
	}
}

func MethodMatchFunc(args ...interface{}) (interface{}, error) {
	name1 := args[0].(string)
	name2 := args[1].(string)

	return MethodMatch(name1, name2), nil
}

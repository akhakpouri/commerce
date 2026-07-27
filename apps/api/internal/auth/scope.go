package auth

type rwScopes struct {
	Read, Write string
}

type userScopes struct {
	Read, Write, Delete string
}

type scopes struct {
	Category, Orders, Payment, Products, Reviews rwScopes
	Users                                        userScopes
}

var Scopes = scopes{
	Category: rwScopes{Read: "category:read", Write: "category:write"},
	Orders:   rwScopes{Read: "orders:read", Write: "orders:write"},
	Payment:  rwScopes{Read: "payment:read", Write: "payment:write"},
	Products: rwScopes{Read: "products:read", Write: "products:write"},
	Reviews:  rwScopes{Read: "reviews:read", Write: "reviews:write"},
	Users:    userScopes{Read: "users:read", Write: "users:write", Delete: "users:delete"},
}

package config

type Auth struct {
	*API
}

type AuthLogin struct {
	*Auth
}

type AuthLogout struct {
	*Auth
}

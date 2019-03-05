package models

type Product string

const (
  XLR Product = "xl-release"
  XLD Product = "xl-deploy"
)

const (
  XLD_LOGIN_TOKEN string = "SESSION_XLD"
  XLR_LOGIN_TOKEN string = "JSESSIONID"
)
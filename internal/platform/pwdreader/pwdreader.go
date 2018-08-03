package pwdreader

type PasswordReader interface {
	ReadPassword(fd int) ([]byte, error)
}

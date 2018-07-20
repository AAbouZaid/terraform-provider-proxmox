package proxmox

func Btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

func Itob(i int) bool {
	if i == 1 {
		return true
	}
	return false
}

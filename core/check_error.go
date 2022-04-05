package core

import "log"

/* 检测错误 */
func CheckErr(err error) {
	if err != nil {
		log.Fatal("\033[1;31m[ERRO] \033[0m", err)
	}
}
func CheckMsg(msg string) {
	log.Fatal("\033[1;31m[ERRO] \033[0m", msg)
}

func CheckErrMsg(err error, msg string) {
	if err != nil {
		log.Fatal("\033[1;31m[ERRO] \033[0m", msg, err)
	}
}

func CheckNoErrMsg(err error, msg string) {
	if err == nil {
		log.Fatal("\033[1;31m[ERRO] \033[0m", msg)
	}
}

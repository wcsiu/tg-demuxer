package tg

// #cgo linux CFLAGS: -I/usr/local/include
// #cgo linux LDFLAGS: -L/usr/local/lib -ltdjson_static -ltdjson_private -ltdclient -ltdcore -ltdactor -ltddb -ltdsqlite -ltdnet -ltdutils -ldl -lm
// #cgo linux LDFLAGS: -L/usr/lib/x86_64-linux-gnu -lssl -lcrypto -lstdc++
// #cgo linux LDFLAGS: -L/lib/x86_64-linux-gnu -lz
// #include <td/telegram/td_json_client.h>
// #include <stdlib.h>
import "C"
import "unsafe"

//Send td_json_client_send()
func Send(client unsafe.Pointer, query string) {
	var q = C.CString(query)
	defer C.free(unsafe.Pointer(q))
	C.td_json_client_send(client, q)
}

//Receive td_json_client_receive()
func Receive(client unsafe.Pointer) string {
	return C.GoString(C.td_json_client_receive(client, 1.0))
}

//CreateClient td_json_client_create()
func CreateClient() unsafe.Pointer {
	return C.td_json_client_create()
}

//DestoryClient td_json_client_destroy()
func DestoryClient(client unsafe.Pointer) {
	C.td_json_client_destroy(client)
}
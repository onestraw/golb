// package controller provides REST API to configure balancer
//
// controller API
//
// - Authentication
// 	Basic HTTP Auth
//
// - Stats
//	GET http://{controller_address}/stats
//
// - List All LB instance
//	GET http://{controller_address}/vs
//
// - Add LB instance
//	POST http://{controller_address}/vs
//	Body {"name":"redis","address":"127.0.0.1:6379"}
//	Example: curl -XPOST -u admin:admin -H 'content-type: application/json' -d '{"name":"redis","address":"127.0.0.1:6379"}' http://127.0.0.1:6587/vs
//
// - Enable LB instance
//	POST http://{controller_address}/vs/{name}
//	Body {"action":"enable"}
//
// - Disable LB instance
//	POST http://{controller_address}/vs/{name}
//	Body {"action":"disable"}
//
// - List pool member of LB instance
//	GET http://{controller_address}/vs/{name}
//
// - Add pool member to LB instance
//	POST http://{controller_address}/vs/{name}/pool
//	Body: {"address":"127.0.0.1:10003","weight":2}
//	Example: curl -XPOST -u admin:admin -H 'content-type: application/json' -d '{"address":"127.0.0.1:10003"}' http://127.0.0.1:6587/vs/web/pool
//
// - Remove pool member from LB instance
//	DELETE http://{controller_address}/vs/{name}/pool
//	Body: {"address":"127.0.0.1:10002"}
//	Example: curl -XDELETE -u admin:admin -H 'content-type: application/json' -d '{"address":"127.0.0.1:10002"}' http://127.0.0.1:6587/vs/web/pool
//
package controller

// package balancer provides service of dispatching requests between server pool
//
// controller API
//
// - Authentication
// 	Basic HTTP Auth
//
// - Stats
//	GET http://{controller_address}/stats
//
// - Enable LB instance
//	POST http://{controller_address}/vs/{name}/enable
//
// - Disable LB instance
//	POST http://{controller_address}/vs/{name}/disable
//
//
package balancer

// Package roundrobin provides smooth weighted round-robin balancing
//
// > For edge case weights like { 5, 1, 1 } we now produce { a, a, b, a, c, a, a }
// > sequence instead of { c, b, a, a, a, a, a } produced previously.
//
// the basic idea is from nginx, refer details in following link
// https://github.com/nginx/nginx/commit/52327e0627f49dbda1e8db695e63a4b0af4448b1
package roundrobin

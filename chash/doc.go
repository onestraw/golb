// package chash provides consistent hash balancing
//
// "Consistent hashing can guarantee that when a cache machine is removed,
// only the objects cached in it will be rehashed; when a new cache machine
// is added, only a fairly few objects will be rehashed."
// https://www.codeproject.com/Articles/56138/Consistent-hashing
//
// consistent hashing with bounded loads
// https://research.googleblog.com/2017/04/consistent-hashing-with-bounded-loads.html
package chash

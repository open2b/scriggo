// Go version: go1.11.5

package net

import original "net"
import "scrigo"
import "reflect"

var Package = scrigo.Package{
	"Addr": reflect.TypeOf((*original.Addr)(nil)).Elem(),
	"AddrError": reflect.TypeOf(original.AddrError{}),
	"Buffers": reflect.TypeOf((original.Buffers)(nil)),
	"CIDRMask": original.CIDRMask,
	"Conn": reflect.TypeOf((*original.Conn)(nil)).Elem(),
	"DNSConfigError": reflect.TypeOf(original.DNSConfigError{}),
	"DNSError": reflect.TypeOf(original.DNSError{}),
	"DefaultResolver": &original.DefaultResolver,
	"Dial": original.Dial,
	"DialIP": original.DialIP,
	"DialTCP": original.DialTCP,
	"DialTimeout": original.DialTimeout,
	"DialUDP": original.DialUDP,
	"DialUnix": original.DialUnix,
	"Dialer": reflect.TypeOf(original.Dialer{}),
	"ErrWriteToConnected": &original.ErrWriteToConnected,
	"Error": reflect.TypeOf((*original.Error)(nil)).Elem(),
	"FileConn": original.FileConn,
	"FileListener": original.FileListener,
	"FilePacketConn": original.FilePacketConn,
	"Flags": reflect.TypeOf(original.Flags(uint(0))),
	"HardwareAddr": reflect.TypeOf((original.HardwareAddr)(nil)),
	"IP": reflect.TypeOf((original.IP)(nil)),
	"IPAddr": reflect.TypeOf(original.IPAddr{}),
	"IPConn": reflect.TypeOf(original.IPConn{}),
	"IPMask": reflect.TypeOf((original.IPMask)(nil)),
	"IPNet": reflect.TypeOf(original.IPNet{}),
	"IPv4": original.IPv4,
	"IPv4Mask": original.IPv4Mask,
	"IPv4allrouter": &original.IPv4allrouter,
	"IPv4allsys": &original.IPv4allsys,
	"IPv4bcast": &original.IPv4bcast,
	"IPv4zero": &original.IPv4zero,
	"IPv6interfacelocalallnodes": &original.IPv6interfacelocalallnodes,
	"IPv6linklocalallnodes": &original.IPv6linklocalallnodes,
	"IPv6linklocalallrouters": &original.IPv6linklocalallrouters,
	"IPv6loopback": &original.IPv6loopback,
	"IPv6unspecified": &original.IPv6unspecified,
	"IPv6zero": &original.IPv6zero,
	"Interface": reflect.TypeOf(original.Interface{}),
	"InterfaceAddrs": original.InterfaceAddrs,
	"InterfaceByIndex": original.InterfaceByIndex,
	"InterfaceByName": original.InterfaceByName,
	"Interfaces": original.Interfaces,
	"InvalidAddrError": reflect.TypeOf(""),
	"JoinHostPort": original.JoinHostPort,
	"Listen": original.Listen,
	"ListenConfig": reflect.TypeOf(original.ListenConfig{}),
	"ListenIP": original.ListenIP,
	"ListenMulticastUDP": original.ListenMulticastUDP,
	"ListenPacket": original.ListenPacket,
	"ListenTCP": original.ListenTCP,
	"ListenUDP": original.ListenUDP,
	"ListenUnix": original.ListenUnix,
	"ListenUnixgram": original.ListenUnixgram,
	"Listener": reflect.TypeOf((*original.Listener)(nil)).Elem(),
	"LookupAddr": original.LookupAddr,
	"LookupCNAME": original.LookupCNAME,
	"LookupHost": original.LookupHost,
	"LookupIP": original.LookupIP,
	"LookupMX": original.LookupMX,
	"LookupNS": original.LookupNS,
	"LookupPort": original.LookupPort,
	"LookupSRV": original.LookupSRV,
	"LookupTXT": original.LookupTXT,
	"MX": reflect.TypeOf(original.MX{}),
	"NS": reflect.TypeOf(original.NS{}),
	"OpError": reflect.TypeOf(original.OpError{}),
	"PacketConn": reflect.TypeOf((*original.PacketConn)(nil)).Elem(),
	"ParseCIDR": original.ParseCIDR,
	"ParseError": reflect.TypeOf(original.ParseError{}),
	"ParseIP": original.ParseIP,
	"ParseMAC": original.ParseMAC,
	"Pipe": original.Pipe,
	"ResolveIPAddr": original.ResolveIPAddr,
	"ResolveTCPAddr": original.ResolveTCPAddr,
	"ResolveUDPAddr": original.ResolveUDPAddr,
	"ResolveUnixAddr": original.ResolveUnixAddr,
	"Resolver": reflect.TypeOf(original.Resolver{}),
	"SRV": reflect.TypeOf(original.SRV{}),
	"SplitHostPort": original.SplitHostPort,
	"TCPAddr": reflect.TypeOf(original.TCPAddr{}),
	"TCPConn": reflect.TypeOf(original.TCPConn{}),
	"TCPListener": reflect.TypeOf(original.TCPListener{}),
	"UDPAddr": reflect.TypeOf(original.UDPAddr{}),
	"UDPConn": reflect.TypeOf(original.UDPConn{}),
	"UnixAddr": reflect.TypeOf(original.UnixAddr{}),
	"UnixConn": reflect.TypeOf(original.UnixConn{}),
	"UnixListener": reflect.TypeOf(original.UnixListener{}),
	"UnknownNetworkError": reflect.TypeOf(""),
}

# Port forwarding rules for http and https
:local destination 10.0.0.11;
/ip firewall nat
add action=dst-nat chain=dstnat dst-port=80    in-interface-list=WAN protocol=tcp to-addresses=$destination to-ports=80
add action=dst-nat chain=dstnat dst-port=443   in-interface-list=WAN protocol=tcp to-addresses=$destination to-ports=443
add action=dst-nat chain=dstnat dst-port=25565 in-interface-list=WAN protocol=tcp to-address=10.0.0.68 to-port=25565

# Drop all port-forwarded packets that aren't coming from cloudflare.
:foreach port in={80; 443} do={
	/ip firewall filter add          \
		action=drop                  \
		chain=forward                \
		connection-nat-state=dstnat  \
		dst-port=$port               \
		protocol=tcp                 \
		src-address-list=!cloudflare \
		in-interface-list=WAN        \
		log=yes                      \
		log-prefix=drop-no-cf        \
		comment="Drop dstnat packets not from cloudflare: $port";
}